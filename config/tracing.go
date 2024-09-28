package config

type TracingConfig struct {
	ServiceName   string              `yaml:"service_name" default:"WebhookX"`
	SamplingRate  float64             `yaml:"sampling_rate" default:"1"`
	Opentelemetry OpenTelemetryConfig `yaml:"opentelemetry"`
}

type OpenTelemetryConfig struct {
	HTTP *OtelHTTP `yaml:"http"`
	GRPC *OtelGPRC `yaml:"grpc"`
}

type OtelHTTP struct {
	Endpoint string            `yaml:"endpoint" default:"http://localhost:4318/v1/traces"`
	Headers  map[string]string `yaml:"headers"`
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
