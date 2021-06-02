package analytics

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

// reNotGoodEnvVarChars is a negated character set of characters that are good
// to appear in environment variables.
var reNotGoodEnvVarChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func toEnvVar(s string) string {
	return strings.ToUpper(reNotGoodEnvVarChars.ReplaceAllString(s, "_"))
}

// environmentVariableEndpointResolver implements endpoints.ResolverFunc by
// reading an environment variable corresponding to the service and region.
// This is how aws-sdk-go supports custom endpoints:
// https://docs.aws.amazon.com/sdk-for-go/api/aws/endpoints/
func environmentVariableEndpointResolver(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
	// e.g., AWS_S3_US_WEST_1_ENDPOINT
	envVar := fmt.Sprintf("AWS_%s_%s_ENDPOINT", toEnvVar(service), toEnvVar(region))
	if e := os.Getenv(envVar); e != "" {
		return endpoints.ResolvedEndpoint{
			URL: e,
		}, nil
	}

	return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
}

var EndpointResolver endpoints.Resolver = endpoints.ResolverFunc(environmentVariableEndpointResolver)
