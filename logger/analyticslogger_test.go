package logger

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	gomock "github.com/golang/mock/gomock"
)

func TestAnalyticsLogger(t *testing.T) {
	tests := []struct {
		name             string
		alc              AnalyticsLoggerConfig
		mockExpectations func(mf *MockFirehoseAPI)
		ops              func(l KayveeLogger)
	}{
		{
			name: "sends one log",
			alc: AnalyticsLoggerConfig{
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
			ops: func(l KayveeLogger) {
				l.InfoD("test-title", M{"foo": "bar"})
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
			al, err := NewAnalyticsLogger(tt.alc)
			if err != nil {
				t.Fatal(err)
			}
			tt.ops(al)
			if err := al.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
