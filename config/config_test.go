package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 RedisConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: RedisConfig{
				Host:     "127.0.0.1",
				Port:     6379,
				Password: "",
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid port",
			cfg: RedisConfig{
				Host:     "127.0.0.1",
				Port:     65536,
				Password: "",
			},
			expectedValidateErr: errors.New("port must be in the range [0, 65535]"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestLogConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 LogConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: LogConfig{
				Level:  LogLevelInfo,
				Format: LogFormatText,
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid level",
			cfg: LogConfig{
				Level:  "",
				Format: LogFormatText,
			},
			expectedValidateErr: errors.New("invalid level: "),
		},
		{
			desc: "invalid level: x",
			cfg: LogConfig{
				Level:  "x",
				Format: LogFormatText,
			},
			expectedValidateErr: errors.New("invalid level: x"),
		},
		{
			desc: "invalid format",
			cfg: LogConfig{
				Level:  "info",
				Format: "",
			},
			expectedValidateErr: errors.New("invalid format: "),
		},
		{
			desc: "invalid format: x",
			cfg: LogConfig{
				Level:  "info",
				Format: "x",
			},
			expectedValidateErr: errors.New("invalid format: x"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestTracingConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 TracingConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: TracingConfig{
				ServiceName:             "WebhookX",
				GlobalAttributes:        map[string]string{},
				CapturedRequestHeaders:  []string{},
				CapturedResponseHeaders: []string{},
				SafeQueryParams:         []string{},
				SamplingRate:            1,
				AddInternals:            false,
				Opentelemetry: &OpenTelemetryConfig{
					HTTP: &OtelHTTP{
						Endpoint: "http://localhost:4318/v1/traces",
						Headers:  map[string]string{},
						TLS:      nil,
					},
					GRPC: &OtelGPRC{
						Endpoint: "localhost:4317",
						Headers:  map[string]string{},
						Insecure: false,
					},
				},
			},
			expectedValidateErr: nil,
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestConfig(t *testing.T) {
	cfg, err := Init()
	assert.Nil(t, err)
	assert.Nil(t, cfg.Validate())
}
