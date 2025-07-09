package analytics

import (
	context "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Clever/kayvee-go/v7/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/firehose/types"
)

//go:generate mockgen -package $GOPACKAGE -source analyticslogger.go -destination mock_firehose.go FirehoseClient

type FirehoseClient interface {
	PutRecordBatch(ctx context.Context, params *firehose.PutRecordBatchInput, optFns ...func(*firehose.Options)) (*firehose.PutRecordBatchOutput, error)
}

// Logger writes to Firehose.
type Logger struct {
	logger.KayveeLogger
	errLogger       logger.KayveeLogger
	fhStream        string
	fhAPI           FirehoseClient
	batch           []types.Record
	batchBytes      int
	maxBatchRecords int
	maxBatchBytes   int
	sendingTicker   *time.Ticker
	done            chan bool
	mu              sync.Mutex
	sendBatchWG     sync.WaitGroup
}

var _ logger.KayveeLogger = &Logger{}
var _ io.WriteCloser = &Logger{}

var ignoredFields = []string{"level", "source", "title", "deploy_env", "wf_id"}

const timeoutForSendingBatches = time.Minute

// firehosePutRecordBatchMaxRecords is an AWS limit.
// https://docs.aws.amazon.com/firehose/latest/APIReference/API_PutRecordBatch.html
const firehosePutRecordBatchMaxRecords = 500

// firehosePutRecordBatchMaxBytes is an AWS limit on total bytes in a PutRecordBatch request.
// https://docs.aws.amazon.com/firehose/latest/APIReference/API_PutRecordBatch.html
const firehosePutRecordBatchMaxBytes = 4000000

// firehosePutRecordBatchMaxTime is a default max time before sending a batch, so that events
// don't get stuck indefinitely. It can be overridden.
const firehosePutRecordBatchMaxTime = 10 * time.Minute

// Config configures things related to collecting analytics.
type Config struct {
	// DBName is the name of the ark db. Either specify this or StreamName.
	DBName string
	// Environment is the name of the environment to point to. Default is _DEPLOY_ENV.
	Environment string
	// StreamName is the name of the Firehose to send to. Either specify this or DBName.
	StreamName string
	// Region is the region where this is running. Defaults to _POD_REGION.
	Region string
	// FirehosePutRecordBatchMaxRecords overrides the default value (500) for the maximum number of records to send in a firehose batch.
	FirehosePutRecordBatchMaxRecords int
	// FirehosePutRecordBatchMaxBytes overrides the default value (4000000) for the maximum number of bytes to send in a firehose batch.
	FirehosePutRecordBatchMaxBytes int
	// FirehosePutRecordBatchMaxTime overrides the default value (10 minutes) for the maximum amount of time between writing an event and sending to the firehose.
	FirehosePutRecordBatchMaxTime time.Duration
	// FirehoseAPI defaults to an API object configured with Region, but can be overriden here.
	FirehoseAPI FirehoseClient
	// ErrLogger is a logger used to make sure errors from goroutines still get surfaced. Defaults to basic logger.Logger
	ErrLogger logger.KayveeLogger
	// VPCEndpoint determines whether to use the VPC endpoint of firehose. Default: true
	VPCEndpoint *bool
}

// New returns a logger that writes to an analytics ark db.
// It takes as input the db name and the ark db config file.
func New(c Config) (*Logger, error) {
	l := logger.New(c.DBName)
	al := &Logger{KayveeLogger: l}
	l.SetOutput(al)
	env, dbname, streamName := c.Environment, c.DBName, c.StreamName
	if dbname != "" && streamName != "" {
		return nil, errors.New("cannot specify both DBName and StreamName in logger config")
	}
	if dbname == "" && streamName == "" {
		return nil, errors.New("must specify either DBName or StreamName in logger config")
	}
	if env == "" {
		if env = os.Getenv("_DEPLOY_ENV"); env == "" {
			return nil, errors.New("env could not be set (either pass in explicit env, or set _DEPLOY_ENV)")
		}
	}
	if dbname != "" {
		al.fhStream = fmt.Sprintf("%s--%s", env, dbname)
	} else {
		al.fhStream = streamName
	}
	vpcEndpoint := true
	if c.VPCEndpoint != nil {
		vpcEndpoint = *c.VPCEndpoint
	}

	if v := c.FirehosePutRecordBatchMaxRecords; v != 0 {
		al.maxBatchRecords = min(v, firehosePutRecordBatchMaxRecords)
	} else {
		al.maxBatchRecords = firehosePutRecordBatchMaxRecords
	}
	if v := c.FirehosePutRecordBatchMaxBytes; v != 0 {
		al.maxBatchBytes = min(v, firehosePutRecordBatchMaxBytes)
	} else {
		al.maxBatchBytes = firehosePutRecordBatchMaxBytes
	}
	if v := c.FirehosePutRecordBatchMaxTime; v > 0 {
		al.sendingTicker = time.NewTicker(v)
	} else {
		al.sendingTicker = time.NewTicker(firehosePutRecordBatchMaxTime)
	}
	al.done = make(chan bool)

	if c.FirehoseAPI != nil {
		al.fhAPI = c.FirehoseAPI
		// make an effort to override endpoint resolver
		if f, ok := c.FirehoseAPI.(*firehose.Client); ok && vpcEndpoint {
			newf := firehose.New(f.Options(), firehose.WithEndpointResolverV2(&endpointResolver{}))
			al.fhAPI = newf
		}
	} else if c.Region != "" {
		awscfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(c.Region))
		if err != nil {
			return nil, fmt.Errorf("error creating firehose client: %v", err)
		}
		al.fhAPI = firehose.NewFromConfig(awscfg, func(o *firehose.Options) {
			if vpcEndpoint {
				firehose.WithEndpointResolverV2(&endpointResolver{})(o)
			}
		})
	} else {
		return nil, errors.New("must provide FirehoseAPI or Region")
	}

	if c.ErrLogger != nil {
		al.errLogger = c.ErrLogger
	} else {
		al.errLogger = logger.New(al.fhStream)
	}

	go func() {
		for {
			select {
			case <-al.done:
				return
			case <-al.sendingTicker.C:
				al.flush()
			}
		}
	}()

	return al, nil
}

// Write a log.
func (al *Logger) Write(bs []byte) (int, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(bs, &m); err != nil {
		return 0, err
	}
	// delete kv-added fields we don't care about. We only want the logger.M values.
	for _, f := range ignoredFields {
		delete(m, f)
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	bs = append(bs, '\n')
	al.mu.Lock()
	al.batchBytes += len(bs)
	al.batch = append(al.batch, types.Record{Data: bs})
	shouldSendBatch := len(al.batch) == al.maxBatchRecords ||
		al.batchBytes > int(0.9*float64(al.maxBatchBytes))
	al.mu.Unlock()

	if shouldSendBatch {
		al.flush()
	}
	return len(bs), nil
}

// flush asynchronously flushes a batch to kinesis
func (al *Logger) flush() {
	al.mu.Lock()
	defer al.mu.Unlock()
	if len(al.batch) > 0 {
		batch := al.batch
		al.batch = nil
		al.batchBytes = 0
		// be careful not to send al.batch, since we will unlock before we finish sending the batch
		al.sendBatchWG.Add(1)
		go func() {
			defer al.sendBatchWG.Done()
			err := sendBatch(batch, al.fhAPI, al.fhStream, time.Now().Add(timeoutForSendingBatches))
			if err != nil {
				al.errLogger.ErrorD("send-batch-error", logger.M{
					"stream": al.fhStream,
					"error":  err.Error(),
				})
			}
		}()
	}
}

// Close flushes all logs to Firehose.
func (al *Logger) Close() error {
	al.sendingTicker.Stop()
	al.done <- true
	al.flush()
	al.sendBatchWG.Wait()
	return nil
}

func sendBatch(batch []types.Record, fhAPI FirehoseClient, fhStream string, timeout time.Time) error {
	// call PutRecordBatch until all records in the batch have been sent successfully
	for time.Now().Before(timeout) {
		ctx, cancel := context.WithDeadline(context.Background(), timeout)
		defer cancel()
		result, err := fhAPI.PutRecordBatch(ctx, &firehose.PutRecordBatchInput{
			DeliveryStreamName: aws.String(fhStream),
			Records:            batch,
		})
		if err != nil {
			return err
		}
		if aws.ToInt32(result.FailedPutCount) == 0 {
			return nil
		}
		// formulate a new batch consisting of the unprocessed items
		newbatch := []types.Record{}
		for i, res := range result.RequestResponses {
			if aws.ToString(res.ErrorCode) == "" {
				continue
			}
			newbatch = append(newbatch, batch[i])
		}
		batch = newbatch
	}
	return fmt.Errorf("timed out sending events: %d remaining", len(batch))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
