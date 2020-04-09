package logger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/eapache/go-resiliency/retrier"
)

//go:generate mockgen -package logger -destination mock_firehose.go github.com/aws/aws-sdk-go/service/firehose/firehoseiface FirehoseAPI

// AnalyticsLogger writes to Firehose instead of the logging pipeline
type AnalyticsLogger struct {
	KayveeLogger
	fhStream        string
	fhAPI           firehoseiface.FirehoseAPI
	batch           []*firehose.Record
	batchBytes      int
	maxBatchRecords int
	maxBatchBytes   int
	mu              sync.Mutex
}

var _ KayveeLogger = &AnalyticsLogger{}
var _ io.WriteCloser = &AnalyticsLogger{}

// firehosePutRecordBatchMaxRecords is an AWS limit.
// https://docs.aws.amazon.com/firehose/latest/APIReference/API_PutRecordBatch.html
const firehosePutRecordBatchMaxRecords = 500

// firehosePutRecordBatchMaxBytes is an AWS limit on total bytes in a PutRecordBatch request.
// https://docs.aws.amazon.com/firehose/latest/APIReference/API_PutRecordBatch.html
const firehosePutRecordBatchMaxBytes = 4000000

// AnalyticsLoggerConfig configures things related to collecting analytics.
type AnalyticsLoggerConfig struct {
	// Environment is the name of the environment to point to. Default is _DEPLOY_ENV.
	Environment string
	// DBName is the name of the ark db.
	DBName string
	// Region is the region where this is running. Defaults to _POD_REGION.
	Region string
	// FirehosePutRecordBatchMaxRecords overrides the default value (500) for the maximum number of records to send in a firehose batch.
	FirehosePutRecordBatchMaxRecords int
	// FirehosePutRecordBatchMaxBytes overrides the default value (4000000) for the maximum number of bytes to send in a firehose batch.
	FirehosePutRecordBatchMaxBytes int
	// FirehoseAPI defaults to an API object configured with Region, but can be overriden here.
	FirehoseAPI firehoseiface.FirehoseAPI
}

// NewAnalyticsLogger returns a logger that writes to an analytics ark db.
// It takes as input the db name and the ark db config file.
func NewAnalyticsLogger(alc AnalyticsLoggerConfig) (*AnalyticsLogger, error) {
	l := New(alc.DBName)
	al := &AnalyticsLogger{KayveeLogger: l}
	l.SetOutput(al)
	env, dbname := alc.Environment, alc.DBName
	if env == "" {
		env = os.Getenv("_DEPLOY_ENV")
	}
	if env == "" {
		return nil, errors.New("_DEPLOY_ENV not set")
	}
	al.fhStream = fmt.Sprintf("%s--%s", env, dbname)

	if v := alc.FirehosePutRecordBatchMaxRecords; v != 0 {
		al.maxBatchRecords = min(v, firehosePutRecordBatchMaxRecords)
	} else {
		al.maxBatchRecords = firehosePutRecordBatchMaxRecords
	}
	if v := alc.FirehosePutRecordBatchMaxBytes; v != 0 {
		al.maxBatchBytes = min(v, firehosePutRecordBatchMaxBytes)
	} else {
		al.maxBatchBytes = firehosePutRecordBatchMaxBytes
	}

	if alc.FirehoseAPI != nil {
		al.fhAPI = alc.FirehoseAPI
	} else if alc.Region != "" {
		al.fhAPI = firehose.New(session.New(&aws.Config{
			Region: aws.String(string(alc.Region)),
		}))
	} else {
		return nil, errors.New("must provide FirehoseAPI or Region")
	}
	return al, nil
}

// Write a log.
func (al *AnalyticsLogger) Write(bs []byte) (int, error) {
	al.mu.Lock()
	defer al.mu.Unlock()
	var m map[string]interface{}
	if err := json.Unmarshal(bs, &m); err != nil {
		return 0, err
	}
	// delete kv-added fields we don't care about. We only want the logger.M values.
	for _, f := range []string{"level", "source", "title", "deploy_env", "wf_id"} {
		delete(m, f)
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	bs = append(bs, '\n')
	al.batchBytes += len(bs)
	al.batch = append(al.batch, &firehose.Record{Data: bs})
	if len(al.batch) == al.maxBatchRecords || al.batchBytes > int(0.9*float64(al.maxBatchBytes)) {
		return len(bs), al.sendBatch()
	}
	return len(bs), nil
}

// Close flushes all logs to Firehose.
func (al *AnalyticsLogger) Close() error {
	al.mu.Lock()
	defer al.mu.Unlock()
	if len(al.batch) > 0 {
		return al.sendBatch()
	}
	return nil
}

func (al *AnalyticsLogger) sendBatch() error {
	defer func() {
		// reset the batch state no matter what, since
		// errors will only get worse if we keep adding to
		// the batch (i.e. we'll exceed record or byte limits)
		al.batch = []*firehose.Record{}
		al.batchBytes = 0
	}()

	// call PutRecordBatch until all records in the batch have been sent successfully
	batch := al.batch
	for {
		var result *firehose.PutRecordBatchOutput
		r := retrier.New(retrier.ExponentialBackoff(5, 100*time.Millisecond), RequestErrorClassifier{})
		if err := r.Run(func() error {
			out, err := al.fhAPI.PutRecordBatch(&firehose.PutRecordBatchInput{
				DeliveryStreamName: aws.String(al.fhStream),
				Records:            batch,
			})
			if err != nil {
				return err
			}
			result = out
			return nil
		}); err != nil {
			return fmt.Errorf("PutRecords: %v", err)
		}
		if aws.Int64Value(result.FailedPutCount) == 0 {
			break
		}
		// formulate a new batch consisting of the unprocessed items
		newbatch := []*firehose.Record{}
		for i, res := range result.RequestResponses {
			if aws.StringValue(res.ErrorCode) != "" {
				newbatch = append(newbatch, batch[i])
			}
		}
		batch = newbatch
	}
	return nil
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
