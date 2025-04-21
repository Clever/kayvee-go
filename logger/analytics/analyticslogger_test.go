package analytics

import (
	"context"
	"testing"

	"github.com/Clever/kayvee-go/v7/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/firehose/types"
	gomock "github.com/golang/mock/gomock"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name             string
		alc              Config
		mockExpectations func(mf *MockFirehoseClient)
		ops              func(l logger.KayveeLogger)
	}{
		{
			name: "sends one log",
			alc: Config{
				Environment: "testenv",
				DBName:      "testdb",
			},
			mockExpectations: func(mf *MockFirehoseClient) {
				mf.EXPECT().PutRecordBatch(context.Background(), &firehose.PutRecordBatchInput{
					DeliveryStreamName: aws.String("testenv--testdb"),
					Records: []types.Record{
						{Data: []byte(`{"foo":"bar"}
`)},
					},
				}).Return(&firehose.PutRecordBatchOutput{FailedPutCount: aws.Int32(0)}, nil)
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
			mf := NewMockFirehoseClient(c)
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
		})
	}
}
