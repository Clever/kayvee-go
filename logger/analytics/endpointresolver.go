package analytics

import (
	context "context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/firehose"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

// reNotGoodEnvVarChars is a negated character set of characters that are good
// to appear in environment variables.
var reNotGoodEnvVarChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func toEnvVar(s string) string {
	return strings.ToUpper(reNotGoodEnvVarChars.ReplaceAllString(s, "_"))
}

type endpointResolver struct{}

func (*endpointResolver) ResolveEndpoint(ctx context.Context, params firehose.EndpointParameters) (smithyendpoints.Endpoint, error) {
	envVar := fmt.Sprintf("AWS_FIREHOSE_%s_ENDPOINT", toEnvVar(*params.Region))
	if e := os.Getenv(envVar); e != "" {
		u, err := url.Parse(e)
		if err != nil {
			return smithyendpoints.Endpoint{}, err
		}
		return smithyendpoints.Endpoint{
			URI: *u,
		}, nil
	}

	return firehose.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
}
