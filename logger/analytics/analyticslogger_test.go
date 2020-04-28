package analytics

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	gomock "github.com/golang/mock/gomock"
	"gopkg.in/Clever/kayvee-go.v6/logger"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name             string
		alc              Config
		mockExpectations func(mf *MockFirehoseAPI)
		ops              func(l logger.KayveeLogger)
	}{
		{
			name: "sends one log",
			alc: Config{
				Environment: "testenv",
				DBName:      "testdb",
			},
			mockExpectations: func(mf *MockFirehoseAPI) {
				mf.EXPECT().PutRecordBatch(&firehose.PutRecordBatchInput{
					DeliveryStreamName: aws.String("testenv--testdb"),
					Records: []*firehose.Record{
						&firehose.Record{Data: []byte(`{"foo":"bar"}
`)},
					},
				}).Return(&firehose.PutRecordBatchOutput{FailedPutCount: aws.Int64(0)}, nil)
			},
			ops: func(l logger.KayveeLogger) {
				l.InfoD("test-title", logger.M{"foo": "bar"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()
			mf := NewMockFirehoseAPI(c)
			if tt.mockExpectations != nil {
				tt.mockExpectations(mf)
			}
			tt.alc.FirehoseAPI = mf
			al, err := New(tt.alc)
			if err != nil {
				t.Fatal(err)
			}
			tt.ops(al)
			al.Close()
			// sleep to make sure goroutines have time to complete
			time.Sleep(time.Millisecond)
		})
	}
}
