package opentelemetry

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"

	"github.com/webhookx-io/webhookx/config"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/encoding/gzip"
)

const TracerName = "github.com/webhookx-io/webhookx"

type OpentelemetryConfig config.OpenTelemetryConfig

func (o *OpentelemetryConfig) Setup(serviceName string, samplingRate float64, attributes map[string]string) (trace.Tracer, io.Closer, error) {
	var err error
	var exporter *otlptrace.Exporter

	if o.Protocol == config.OtlpProtocolHTTP {
		exporter, err = setupHTTPExporter(o)
	} else if o.Protocol == config.OtlpProtocolGRPC {
		exporter, err = setupGRPCExporter(o)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup exporter: %w", err)
	}

	attr := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(config.VERSION),
	}

	for k, v := range attributes {
		attr = append(attr, attribute.String(k, v))
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(attr...),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(samplingRate)),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())
	return tracerProvider.Tracer(TracerName), &tpCloser{provider: tracerProvider}, err
}

func setupHTTPExporter(c *OpentelemetryConfig) (*otlptrace.Exporter, error) {
	endpoint, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid collector endpoint %q: %w", c.Endpoint, err)
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint.Host),
		otlptracehttp.WithHeaders(c.Headers),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}

	if endpoint.Scheme == "http" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if endpoint.Path != "" {
		opts = append(opts, otlptracehttp.WithURLPath(endpoint.Path))
	}

	return otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
}

func setupGRPCExporter(c *OpentelemetryConfig) (*otlptrace.Exporter, error) {
	host, port, err := net.SplitHostPort(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid collector endpoint %q: %w", c.Endpoint, err)
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%s", host, port)),
		otlptracegrpc.WithHeaders(c.Headers),
		otlptracegrpc.WithCompressor(gzip.Name),
		otlptracegrpc.WithInsecure(),
	}

	return otlptrace.New(context.Background(), otlptracegrpc.NewClient(opts...))
}

type tpCloser struct {
	provider *sdktrace.TracerProvider
}

func (t *tpCloser) Close() error {
	if t == nil {
		return nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()

	return t.provider.Shutdown(ctx)
}
