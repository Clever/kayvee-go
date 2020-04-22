package analytics

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/eapache/go-resiliency/retrier"
	"gopkg.in/Clever/kayvee-go.v6/logger"
)

//go:generate mockgen -package $GOPACKAGE -destination mock_firehose.go github.com/aws/aws-sdk-go/service/firehose/firehoseiface FirehoseAPI

// Logger writes to Firehose instead of the logging pipeline
type Logger struct {
	logger.KayveeLogger
	fhStream string
	fhAPI    firehoseiface.FirehoseAPI
}

var _ logger.KayveeLogger = &Logger{}
var _ io.Writer = &Logger{}

var ignoredFields = []string{"level", "source", "title", "deploy_env", "wf_id"}

// Config configures things related to collecting analytics.
type Config struct {
	// DBName is the name of the ark db.
	DBName string
	// Environment is the name of the environment to point to. Default is _DEPLOY_ENV.
	Environment string
	// Region is the region where this is running. Defaults to _POD_REGION.
	Region *string
	// FirehoseAPI defaults to an API object configured with Region, but can be overriden here.
	FirehoseAPI firehoseiface.FirehoseAPI
}

// New returns a logger that writes to an analytics ark db.
// It takes as input the db name and the ark db config file.
func New(c Config) (*Logger, error) {
	l := logger.New(c.DBName)
	al := &Logger{KayveeLogger: l}
	l.SetOutput(al)
	env, dbname := c.Environment, c.DBName
	if env == "" {
		env = os.Getenv("_DEPLOY_ENV")
		if env == "" {
			return nil, errors.New("env could not be set (either pass in explicit env, or set _DEPLOY_ENV)")
		}
	}
	al.fhStream = fmt.Sprintf("%s--%s", env, dbname)

	if c.FirehoseAPI != nil {
		al.fhAPI = c.FirehoseAPI
	} else if c.Region != nil {
		sess, err := session.NewSession(&aws.Config{
			Region: c.Region,
		})
		if err != nil {
			return nil, errors.New("unable to create AWS session")
		}
		al.fhAPI = firehose.New(sess)
	} else {
		return nil, errors.New("must provide FirehoseAPI or Region")
	}
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
	r := retrier.New(retrier.ExponentialBackoff(5, 100*time.Millisecond), RequestErrorClassifier{})
	if err := r.Run(func() error {
		_, err := al.fhAPI.PutRecord(&firehose.PutRecordInput{
			DeliveryStreamName: aws.String(al.fhStream),
			Record:             &firehose.Record{Data: bs},
		})
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 0, fmt.Errorf("PutRecords: %v", err)
	}
	return len(bs), nil
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
