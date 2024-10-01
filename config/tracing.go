package config

import "github.com/webhookx-io/webhookx/pkg/types"

type TracingConfig struct {
	ServiceName             string            `yaml:"service_name" default:"WebhookX"`
	GlobalAttributes        map[string]string `yaml:"global_attributes"`
	CapturedRequestHeaders  []string          `yaml:"captured_request_headers"`
	CapturedResponseHeaders []string          `yaml:"captured_response_headers"`
	SafeQueryParams         []string          `yaml:"safe_query_params"`
	SamplingRate            float64           `yaml:"sampling_rate" default:"1"`
	AddInternals            bool              `yaml:"add_internals" default:"false"`

	Opentelemetry *OpenTelemetryConfig `yaml:"opentelemetry"`
}

type OpenTelemetryConfig struct {
	HTTP *OtelHTTP `yaml:"http"`
	GRPC *OtelGPRC `yaml:"grpc"`
}

type OtelHTTP struct {
	Endpoint string            `yaml:"endpoint" default:"http://localhost:4318/v1/traces"`
	Headers  map[string]string `yaml:"headers"`
	TLS      *types.ClientTLS  `yaml:"tls"`
}

type OtelGPRC struct {
	Endpoint string            `yaml:"endpoint" default:"localhost:4317"`
	Headers  map[string]string `yaml:"headers"`
	Insecure bool              `yaml:"insecure" default:"false"`
}

func (cfg TracingConfig) Validate() error {
	// TODO
	return nil
}
