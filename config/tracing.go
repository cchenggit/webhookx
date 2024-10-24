package config

import (
	"errors"
	"fmt"
	"slices"
)

type TracingConfig struct {
	Enabled                 bool                 `yaml:"enabled" default:"false"`
	Attributes              map[string]string    `yaml:"attributes"`
	Opentelemetry           *OpenTelemetryConfig `yaml:"opentelemetry"`
	ServiceName             string               `yaml:"service_name" default:"WebhookX"`
	CapturedRequestHeaders  []string             `yaml:"captured_request_headers"`
	CapturedResponseHeaders []string             `yaml:"captured_response_headers"`
	SafeQueryParams         []string             `yaml:"safe_query_params"`
	SamplingRate            float64              `yaml:"sampling_rate" default:"1.0"`
}

type OpenTelemetryConfig struct {
	Protocol OtlpProtocol      `yaml:"protocol" envconfig:"PROTOCOL" default:"http/protobuf"`
	Endpoint string            `yaml:"endpoint" envconfig:"ENDPOINT" default:"http://localhost:4318/v1/metrics"`
	Headers  map[string]string `yaml:"headers,omitempty"`
}

func (cfg TracingConfig) Validate() error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.SamplingRate > 1 || cfg.SamplingRate < 0 {
		return errors.New("invalid sampling rate, must be [0,1]")
	}
	if cfg.Opentelemetry == nil {
		return errors.New("opentelemetry is required")
	}
	if !slices.Contains([]OtlpProtocol{OtlpProtocolGRPC, OtlpProtocolHTTP}, cfg.Opentelemetry.Protocol) {
		return fmt.Errorf("invalid protocol: %s", cfg.Opentelemetry.Protocol)
	}
	return nil
}
