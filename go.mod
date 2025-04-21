module github.com/Clever/kayvee-go/v7

go 1.24

require (
	github.com/Clever/wag/logging/wagclientlogger v0.0.0-20220916194010-36f974d66e08
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.27.0
	github.com/aws/aws-sdk-go-v2/service/firehose v1.37.4
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.33.3
	github.com/aws/smithy-go v1.22.2
	github.com/eapache/go-resiliency v1.3.0
	github.com/golang/mock v1.6.0
	github.com/stretchr/testify v1.8.0
	github.com/xeipuuv/gojsonschema v1.2.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.23.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/sys v0.0.0-20220829200755-d48e67d00261 // indirect
	golang.org/x/tools v0.1.1 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// For logr (indirect dep of otel), it uses strconv.FormatComplex, which
replace github.com/go-openapi/validate => github.com/go-openapi/validate v0.0.0-20180703152151-9a6e517cddf1 // pre-modules tag 0.15.0

tool github.com/golang/mock/mockgen
