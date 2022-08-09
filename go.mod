module github.com/Clever/kayvee-go/v7

go 1.16

require (
	github.com/aws/aws-sdk-go v1.30.6
	github.com/eapache/go-resiliency v1.2.0
	github.com/golang/mock v1.6.0
	github.com/stretchr/testify v1.7.1
	github.com/xeipuuv/gojsonpointer v0.0.0-20170225233418-6fe8760cad35 // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20150808065054-e02fc20de94c // indirect
	github.com/xeipuuv/gojsonschema v0.0.0-20171025060643-212d8a0df7ac
	go.opentelemetry.io/otel v1.9.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.31.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.26.0
	go.opentelemetry.io/otel/metric v0.31.0
	go.opentelemetry.io/otel/sdk v1.9.0
	go.opentelemetry.io/otel/sdk/metric v0.31.0
	golang.org/x/sys v0.0.0-20220808155132-1c4a2a72c664 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v2 v2.2.3
)

// For logr (indirect dep of otel), it uses strconv.FormatComplex, which
replace github.com/go-openapi/validate => github.com/go-openapi/validate v0.0.0-20180703152151-9a6e517cddf1 // pre-modules tag 0.15.0
