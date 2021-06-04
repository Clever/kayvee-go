package kinesisstream

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/Clever/kayvee-go/logger/analytics"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/eapache/go-resiliency/retrier"
	"gopkg.in/Clever/kayvee-go.v6/logger"
)

//go:generate mockgen -package $GOPACKAGE -destination mock_kinesis.go github.com/aws/aws-sdk-go/service/kinesis/kinesisiface KinesisAPI

// Logger writes to Kinesis.
type Logger struct {
	logger.KayveeLogger
	errLogger       logger.KayveeLogger
	kinesisStream   string
	kinesisAPI      kinesisiface.KinesisAPI
	batch           []*kinesis.PutRecordsRequestEntry
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
	// KinesisAPI defaults to an API object configured with Region, but can be overriden here.
	KinesisAPI kinesisiface.KinesisAPI
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

	if c.KinesisAPI != nil {
		// make an effort to override endpoint resolver
		if k, ok := c.KinesisAPI.(*kinesis.Kinesis); ok {
			k.Client.Config.EndpointResolver = analytics.EndpointResolver
			ksl.kinesisAPI = k
		} else {
			ksl.kinesisAPI = c.KinesisAPI
		}
	} else if c.Region != "" {
		config := aws.NewConfig().WithRegion(c.Region).WithEndpointResolver(analytics.EndpointResolver)
		sess, err := session.NewSession(config)
		if err != nil {
			return nil, fmt.Errorf("error creating kinesis client: %v", err)
		}
		ksl.kinesisAPI = kinesis.New(sess)
	} else {
		return nil, errors.New("must provide KinesisAPI or Region")
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
	ksl.batch = append(ksl.batch, &kinesis.PutRecordsRequestEntry{
		Data:         bs,
		PartitionKey: &partitionKey,
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
func (ksl *Logger) flush() error {
	ksl.mu.Lock()
	defer ksl.mu.Unlock()
	if len(ksl.batch) > 0 {
		batch := ksl.batch
		ksl.batch = nil
		ksl.batchBytes = 0
		// be careful not to send ksl.batch, since we will unlock before we finish sending the batch
		ksl.sendBatchWG.Add(1)
		go func() {
			err := sendBatch(batch, ksl.kinesisAPI, ksl.kinesisStream, time.Now().Add(timeoutForSendingBatches))
			ksl.sendBatchWG.Done()
			if err != nil {
				ksl.errLogger.ErrorD("send-batch-error", logger.M{
					"stream": ksl.kinesisStream,
					"error":  err.Error(),
				})
			}
		}()
	}
	return nil
}

// Close flushes all logs to Kinesis.
func (ksl *Logger) Close() error {
	ksl.sendingTicker.Stop()
	ksl.done <- true
	ksl.flush()
	ksl.sendBatchWG.Wait()
	return nil
}

func sendBatch(batch []*kinesis.PutRecordsRequestEntry, kinesisAPI kinesisiface.KinesisAPI, kinesisStream string, timeout time.Time) error {
	// call PutRecordBatch until all records in the batch have been sent successfully
	for time.Now().Before(timeout) {
		var result *kinesis.PutRecordsOutput
		r := retrier.New(retrier.ExponentialBackoff(5, 100*time.Millisecond), RequestErrorClassifier{})
		if err := r.Run(func() error {
			out, err := kinesisAPI.PutRecords(&kinesis.PutRecordsInput{
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
		if aws.Int64Value(result.FailedRecordCount) == 0 {
			return nil
		}
		// formulate a new batch consisting of the unprocessed items
		newbatch := []*kinesis.PutRecordsRequestEntry{}
		for i, res := range result.Records {
			if aws.StringValue(res.ErrorCode) == "" {
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
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "RequestError" {
		return retrier.Retry
	}
	return retrier.Fail
}
