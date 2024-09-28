package tracing

import (
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/tracing/opentelemetry"
)

func Setup(cfg config.TracingConfig) error {
	return opentelemetry.Setup(cfg)
}
