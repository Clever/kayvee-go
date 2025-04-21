package kinesisstream

import (
	"testing"

	"github.com/Clever/kayvee-go/v7/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	gomock "github.com/golang/mock/gomock"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name             string
		klc              Config
		mockExpectations func(mk *MockKinesisClient)
		ops              func(l logger.KayveeLogger)
	}{
		{
			name: "sends one log",
			klc: Config{
				Environment: "testenv",
				DBName:      "testdb",
			},
			mockExpectations: func(mk *MockKinesisClient) {
				mk.EXPECT().PutRecords(gomock.Any(), &kinesis.PutRecordsInput{
					StreamName: aws.String("testenv--testdb"),
					Records: []types.PutRecordsRequestEntry{
						{
							Data:         []byte(`{"foo":"bar"}` + "\n"),
							PartitionKey: aws.String("1"),
						},
					},
				}).Return(&kinesis.PutRecordsOutput{FailedRecordCount: aws.Int32(0)}, nil)
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
			mk := NewMockKinesisClient(c)
			if tt.mockExpectations != nil {
				tt.mockExpectations(mk)
			}
			tt.klc.KinesisClient = mk
			kl, err := New(tt.klc)
			if err != nil {
				t.Fatal(err)
			}
			tt.ops(kl)
			kl.Close()
		})
	}
}
