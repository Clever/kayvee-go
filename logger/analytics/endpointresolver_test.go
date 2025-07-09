package analytics

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/firehose"
)

func mustUrlParse(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func Test_endpointResolver_ResolveEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		want    url.URL
		wantErr bool
		setEnv  map[string]string
	}{
		{
			name:    "default endpoint",
			region:  "us-west-2",
			want:    *mustUrlParse("https://firehose.us-west-2.amazonaws.com"),
			wantErr: false,
		},
		{
			name:    "default endpoint for us-west-1",
			region:  "us-west-1",
			want:    *mustUrlParse("https://firehose.us-west-1.amazonaws.com"),
			wantErr: false,
			setEnv: map[string]string{
				"AWS_FIREHOSE_US_WEST_2_ENDPOINT": "https://vpce-0123456789abcdefg-hijklmno.firehose.us-west-2.vpce.amazonaws.com",
			},
		},
		{
			name:    "override",
			region:  "us-west-2",
			want:    *mustUrlParse("https://vpce-0123456789abcdefg-hijklmno.firehose.us-west-2.vpce.amazonaws.com"),
			wantErr: false,
			setEnv: map[string]string{
				"AWS_FIREHOSE_US_WEST_2_ENDPOINT": "https://vpce-0123456789abcdefg-hijklmno.firehose.us-west-2.vpce.amazonaws.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.setEnv {
				t.Setenv(k, v)
			}
			e := &endpointResolver{}
			got, err := e.ResolveEndpoint(t.Context(), firehose.EndpointParameters{Region: &tt.region})
			if (err != nil) != tt.wantErr {
				t.Errorf("endpointResolver.ResolveEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.URI, tt.want) {
				t.Errorf("endpointResolver.ResolveEndpoint() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
