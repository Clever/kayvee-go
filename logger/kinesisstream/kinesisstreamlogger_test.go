package kinesisstream

import (
	"testing"

	"github.com/Clever/kayvee-go/v8/logger"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	gomock "github.com/golang/mock/gomock"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name             string
		klc              Config
		mockExpectations func(mk *MockKinesisAPI)
		ops              func(l logger.KayveeLogger)
	}{
		{
			name: "sends one log",
			klc: Config{
				Environment: "testenv",
				DBName:      "testdb",
			},
			mockExpectations: func(mk *MockKinesisAPI) {
				mk.EXPECT().PutRecords(&kinesis.PutRecordsInput{
					StreamName: aws.String("testenv--testdb"),
					Records: []*kinesis.PutRecordsRequestEntry{
						{
							Data: []byte(`{"foo":"bar"}
`),
							PartitionKey: aws.String("1"),
						},
					},
				}).Return(&kinesis.PutRecordsOutput{FailedRecordCount: aws.Int64(0)}, nil)
			},
			ops: func(l logger.KayveeLogger) {
				l.InfoD("test-title", logger.M{"foo": "bar", "partition_key": "1"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()
			mk := NewMockKinesisAPI(c)
			if tt.mockExpectations != nil {
				tt.mockExpectations(mk)
			}
			tt.klc.KinesisAPI = mk
			kl, err := New(tt.klc)
			if err != nil {
				t.Fatal(err)
			}
			tt.ops(kl)
			kl.Close()
		})
	}
}
