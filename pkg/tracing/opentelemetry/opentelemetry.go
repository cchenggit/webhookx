package opentelemetry

import (
	"context"
	"fmt"
	"github.com/webhookx-io/webhookx/config"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/encoding/gzip"
)

// Config provides configuration settings for the open-telemetry tracer.
//type Config struct {
//	GRPC *types.OtelGRPC `description:"gRPC configuration for the OpenTelemetry collector." json:"grpc,omitempty" toml:"grpc,omitempty" yaml:"grpc,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
//	HTTP *types.OtelHTTP `description:"HTTP configuration for the OpenTelemetry collector." json:"http,omitempty" toml:"http,omitempty" yaml:"http,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
//}
//
//// Setup sets up the tracer.
//func (c *Config) Setup(serviceName string, sampleRate float64, globalAttributes map[string]string) (trace.Tracer, io.Closer, error) {
//
//	log.Debug().Msg("OpenTelemetry tracer configured")
//
//	return tracerProvider.Tracer("github.com/traefik/traefik"), &tpCloser{provider: tracerProvider}, err
//}
//
//// tpCloser converts a TraceProvider into an io.Closer.
//type tpCloser struct {
//	provider *sdktrace.TracerProvider
//}
//
//func (t *tpCloser) Close() error {
//	if t == nil {
//		return nil
//	}
//
//	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
//	defer cancel()
//
//	return t.provider.Shutdown(ctx)
//}

func setupHTTPExporter(cfg *config.OtelHTTP) (*otlptrace.Exporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Endpoint),
		otlptracehttp.WithHeaders(cfg.Headers),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}
	return otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
}

func setupGRPCExporter(cfg *config.OtelGPRC) (*otlptrace.Exporter, error) {
	// FIXME grpc exporter does not work!

	//host, port, err := net.SplitHostPort(cfg.Endpoint)
	//if err != nil {
	//	return nil, fmt.Errorf("invalid collector endpoint %q: %w", c.GRPC.Endpoint, err)
	//}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithHeaders(cfg.Headers),
		otlptracegrpc.WithCompressor(gzip.Name),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	//
	//if c.GRPC.TLS != nil {
	//	tlsConfig, err := c.GRPC.TLS.CreateTLSConfig(context.Background())
	//	if err != nil {
	//		return nil, fmt.Errorf("creating TLS client config: %w", err)
	//	}
	//
	//	opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	//}

	return otlptrace.New(context.Background(), otlptracegrpc.NewClient(opts...))
}

func Setup(cfg config.TracingConfig) error {
	var err error
	var exporter *otlptrace.Exporter

	if cfg.Opentelemetry.HTTP != nil {
		exporter, err = setupHTTPExporter(cfg.Opentelemetry.HTTP)
	} else {
		exporter, err = setupGRPCExporter(cfg.Opentelemetry.GRPC)
	}

	if err != nil {
		return fmt.Errorf("failed to setup exporter: %w", err)
	}

	attr := []attribute.KeyValue{
		semconv.ServiceNameKey.String("WebhookX"),
		semconv.ServiceVersionKey.String(config.VERSION),
	}

	//for k, v := range globalAttributes {
	//	attr = append(attr, attribute.String(k, v))
	//}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(attr...), // Add custom reso
		// TODO
		resource.WithFromEnv(),      // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(), // Discover and provide information about the OpenTelemetry SDK used.
		resource.WithProcess(),      // Discover and provide process information.
		resource.WithOS(),           // Discover and provide OS information.
		resource.WithContainer(),    // Discover and provide container information.
		resource.WithHost(),         // Discover and provide host information.
	)
	if err != nil {
		return fmt.Errorf("failed to build resource: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(cfg.SamplingRate)),
		trace.WithResource(res),
		trace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	return nil
}
