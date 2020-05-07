package analytics

import (
	"os"
	"testing"
)

func Test_environmentVariableEndpointResolver(t *testing.T) {
	type args struct {
		service string
		region  string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
		setEnv  map[string]string
	}{
		{
			name:    "default endpoint",
			args:    args{service: "firehose", region: "us-west-2"},
			want:    "https://firehose.us-west-2.amazonaws.com",
			wantErr: false,
		},
		{
			name:    "override",
			args:    args{service: "firehose", region: "us-west-2"},
			want:    "https://vpce-0123456789abcdefg-hijklmno.firehose.us-west-2.vpce.amazonaws.com",
			wantErr: false,
			setEnv: map[string]string{
				"AWS_FIREHOSE_US_WEST_2_ENDPOINT": "https://vpce-0123456789abcdefg-hijklmno.firehose.us-west-2.vpce.amazonaws.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv != nil {
				for k, v := range tt.setEnv {
					os.Setenv(k, v)
					defer os.Unsetenv(k)
				}
			}
			got, err := environmentVariableEndpointResolver(tt.args.service, tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("environmentVariableEndpointResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.URL != tt.want {
				t.Errorf("environmentVariableEndpointResolver() = '%s', want '%s'", got.URL, tt.want)
			}
		})
	}
}
