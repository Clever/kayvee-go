package analytics

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// reNotGoodEnvVarChars is a negated character set of characters that are good
// to appear in environment variables.
var reNotGoodEnvVarChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func toEnvVar(s string) string {
	return strings.ToUpper(reNotGoodEnvVarChars.ReplaceAllString(s, "_"))
}

// environmentVariableEndpointResolver implements aws.EndpointResolverWithOptions by
// reading an environment variable corresponding to the service and region.
func environmentVariableEndpointResolver(service, region string, options ...interface{}) (aws.Endpoint, error) {
	// e.g., AWS_S3_US_WEST_1_ENDPOINT
	envVar := fmt.Sprintf("AWS_%s_%s_ENDPOINT", toEnvVar(service), toEnvVar(region))
	if e := os.Getenv(envVar); e != "" {
		return aws.Endpoint{
			URL: e,
		}, nil
	}

	// Return default endpoint if no custom endpoint is set
	return aws.Endpoint{
		URL: fmt.Sprintf("https://%s.%s.amazonaws.com", service, region),
	}, nil
}

// EndpointResolver is used to override the endpoints that AWS clients use. In
// particular for reducing networking costs for cross-region traffic, we sometimes
// use a VPC endpoint rather than going through the public internet and a NAT Gateway
var EndpointResolver aws.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(environmentVariableEndpointResolver)
