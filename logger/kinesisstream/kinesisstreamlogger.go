package kinesisstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/Clever/kayvee-go/v7/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/aws/smithy-go"
	"github.com/eapache/go-resiliency/retrier"
)

// KinesisClient is the interface for AWS Kinesis operations
type KinesisClient interface {
	PutRecords(ctx context.Context, params *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error)
}

//go:generate mockgen -package kinesisstream -destination mock_kinesis.go -source kinesisstreamlogger.go KinesisClient

// Logger writes to Kinesis.
type Logger struct {
	logger.KayveeLogger
	errLogger       logger.KayveeLogger
	kinesisStream   string
	kinesisClient   KinesisClient
	batch           []types.PutRecordsRequestEntry
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

// Logs with partition_key specified will use that for deciding which shard to send to.
// Otherwise the partition key will be generated randomly.
const partitionKeyFieldName = "partition_key"

const timeoutForSendingBatches = time.Minute

// kinesisPutRecordBatchMaxRecords is an AWS limit.
// https://docs.aws.amazon.com/kinesis/latest/APIReference/API_PutRecords.html
const kinesisPutRecordBatchMaxRecords = 500

// kinesisPutRecordBatchMaxBytes is an AWS limit on total bytes in a PutRecordBatch request.
// https://docs.aws.amazon.com/kinesis/latest/APIReference/API_PutRecords.html
const kinesisPutRecordBatchMaxBytes = 5000000

// kinesisPutRecordBatchMaxTime is a default max time before sending a batch, so that events
// don't get stuck indefinitely. It can be overridden.
const kinesisPutRecordBatchMaxTime = 10 * time.Minute

// Config configures things related to collecting analytics.
type Config struct {
	// DBName is the name of the ark db. Either specify this or StreamName.
	DBName string
	// Environment is the name of the environment to point to. Default is _DEPLOY_ENV.
	Environment string
	// StreamName is the name of the Kinesis stream to send to. Either specify this or DBName.
	StreamName string
	// Region is the region where this is running. Defaults to _POD_REGION.
	Region string
	// KinesisPutRecordBatchMaxRecords overrides the default value (500) for the maximum number of records to send in a batch.
	KinesisPutRecordBatchMaxRecords int
	// KinesisPutRecordBatchMaxBytes overrides the default value (5000000) for the maximum number of bytes to send in as batch.
	KinesisPutRecordBatchMaxBytes int
	// KinesisPutRecordBatchMaxTime overrides the default value (10 minutes) for the maximum amount of time between writing an event and sending to the stream.
	KinesisPutRecordBatchMaxTime time.Duration
	// KinesisClient defaults to a client configured with Region, but can be overriden here.
	KinesisClient KinesisClient
	// ErrLogger is a logger used to make sure errors from goroutines still get surfaced. Defaults to basic logger.Logger
	ErrLogger logger.KayveeLogger
}

// New returns a logger that writes to an analytics ark db.
// It takes as input the db name and the ark db config file.
func New(c Config) (*Logger, error) {
	l := logger.New(c.DBName)
	ksl := &Logger{KayveeLogger: l}
	l.SetOutput(ksl)
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
		ksl.kinesisStream = fmt.Sprintf("%s--%s", env, dbname)
	} else {
		ksl.kinesisStream = streamName
	}

	if v := c.KinesisPutRecordBatchMaxRecords; v != 0 {
		ksl.maxBatchRecords = min(v, kinesisPutRecordBatchMaxRecords)
	} else {
		ksl.maxBatchRecords = kinesisPutRecordBatchMaxRecords
	}
	if v := c.KinesisPutRecordBatchMaxBytes; v != 0 {
		ksl.maxBatchBytes = min(v, kinesisPutRecordBatchMaxBytes)
	} else {
		ksl.maxBatchBytes = kinesisPutRecordBatchMaxBytes
	}
	if v := c.KinesisPutRecordBatchMaxTime; v > 0 {
		ksl.sendingTicker = time.NewTicker(v)
	} else {
		ksl.sendingTicker = time.NewTicker(kinesisPutRecordBatchMaxTime)
	}
	ksl.done = make(chan bool)

	if c.KinesisClient != nil {
		ksl.kinesisClient = c.KinesisClient
	} else if c.Region != "" {
		cfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(c.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating kinesis client: %v", err)
		}
		ksl.kinesisClient = kinesis.NewFromConfig(cfg)
	} else {
		return nil, errors.New("must provide KinesisClient or Region")
	}

	if c.ErrLogger != nil {
		ksl.errLogger = c.ErrLogger
	} else {
		ksl.errLogger = logger.New(ksl.kinesisStream)
	}

	go func() {
		for {
			select {
			case <-ksl.done:
				return
			case <-ksl.sendingTicker.C:
				ksl.flush()
			}
		}
	}()

	return ksl, nil
}

// Write a log.
func (ksl *Logger) Write(bs []byte) (int, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(bs, &m); err != nil {
		return 0, err
	}
	// delete kv-added fields we don't care about. We only want the logger.M values.
	for _, f := range ignoredFields {
		delete(m, f)
	}
	partitionKey, ok := m[partitionKeyFieldName].(string)
	delete(m, partitionKeyFieldName)
	if !ok {
		partitionKey = fmt.Sprintf("%d", rand.Int())
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	bs = append(bs, '\n')
	ksl.mu.Lock()
	ksl.batchBytes += len(bs)
	ksl.batch = append(ksl.batch, types.PutRecordsRequestEntry{
		Data:         bs,
		PartitionKey: aws.String(partitionKey),
	})
	shouldSendBatch := len(ksl.batch) == ksl.maxBatchRecords ||
		ksl.batchBytes > int(0.9*float64(ksl.maxBatchBytes))
	ksl.mu.Unlock()

	if shouldSendBatch {
		ksl.flush()
	}
	return len(bs), nil
}

// flush asynchronously flushes a batch to kinesis
func (ksl *Logger) flush() {
	ksl.mu.Lock()
	defer ksl.mu.Unlock()
	if len(ksl.batch) > 0 {
		batch := ksl.batch
		ksl.batch = nil
		ksl.batchBytes = 0
		// be careful not to send ksl.batch, since we will unlock before we finish sending the batch
		ksl.sendBatchWG.Add(1)
		go func() {
			defer ksl.sendBatchWG.Done()
			err := sendBatch(batch, ksl.kinesisClient, ksl.kinesisStream, time.Now().Add(timeoutForSendingBatches))
			if err != nil {
				ksl.errLogger.ErrorD("send-batch-error", logger.M{
					"stream": ksl.kinesisStream,
					"error":  err.Error(),
				})
			}
		}()
	}
}

// Close flushes all logs to Kinesis.
func (ksl *Logger) Close() error {
	ksl.sendingTicker.Stop()
	ksl.done <- true
	ksl.flush()
	ksl.sendBatchWG.Wait()
	return nil
}

func sendBatch(batch []types.PutRecordsRequestEntry, kinesisClient KinesisClient, kinesisStream string, timeout time.Time) error {
	// call PutRecordBatch until all records in the batch have been sent successfully
	for time.Now().Before(timeout) {
		var result *kinesis.PutRecordsOutput
		r := retrier.New(retrier.ExponentialBackoff(5, 100*time.Millisecond), RequestErrorClassifier{})
		if err := r.Run(func() error {
			out, err := kinesisClient.PutRecords(context.Background(), &kinesis.PutRecordsInput{
				StreamName: aws.String(kinesisStream),
				Records:    batch,
			})
			if err != nil {
				return err
			}
			result = out
			return nil
		}); err != nil {
			return err
		}
		if *result.FailedRecordCount == int32(0) {
			return nil
		}
		// formulate a new batch consisting of the unprocessed items
		newbatch := []types.PutRecordsRequestEntry{}
		for i, res := range result.Records {
			if res.ErrorCode == nil {
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

// RequestErrorClassifier corrects for AWS SDK's lack of automatic retry on
// "RequestError: connection reset by peer"
type RequestErrorClassifier struct{}

var _ retrier.Classifier = RequestErrorClassifier{}

// Classify the error.
func (RequestErrorClassifier) Classify(err error) retrier.Action {
	if err == nil {
		return retrier.Succeed
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "RequestError" {
		return retrier.Retry
	}
	return retrier.Fail
}
