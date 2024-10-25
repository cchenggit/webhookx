package tracing

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/tracing/opentelemetry"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func Setup(conf *config.TracingConfig) (*Tracer, io.Closer, error) {
	if !conf.Enabled {
		return nil, nil, nil
	}
	tracer, closer, err := NewTracing(conf)
	if err != nil {
		zap.S().Warnf("Unable to create tracer:%v", err)
		return nil, nil, err
	}

	return tracer, closer, nil
}

type Backend interface {
	Setup(serviceName string, samplingRate float64, attributes map[string]string) (trace.Tracer, io.Closer, error)
}

func NewTracing(conf *config.TracingConfig) (*Tracer, io.Closer, error) {
	if conf == nil {
		return nil, nil, nil
	}
	var backend Backend

	if conf.Opentelemetry != nil {
		otelConfig := opentelemetry.OpentelemetryConfig(*conf.Opentelemetry)
		backend = &otelConfig
	}

	if backend == nil {
		zap.S().Debug("Could not initialize tracing, using OpenTelemetry by default")
		defaultBackend := &opentelemetry.OpentelemetryConfig{}
		backend = defaultBackend
	}

	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	tr, closer, err := backend.Setup(conf.ServiceName, conf.SamplingRate, conf.Attributes)
	if err != nil {
		return nil, nil, err
	}

	return NewTracer(tr, conf.CapturedRequestHeaders, conf.CapturedResponseHeaders, conf.SafeQueryParams), closer, nil
}

func TracerFromContext(ctx context.Context) *Tracer {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	if span != nil && span.TracerProvider() != nil {
		tracer := span.TracerProvider().Tracer(opentelemetry.TracerName)
		if tracer, ok := tracer.(*Tracer); ok {
			return tracer
		}

		return nil
	}

	return nil
}

func ExtractCarrierIntoContext(ctx context.Context, headers http.Header) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, propagation.HeaderCarrier(headers))
}

func InjectContextIntoCarrier(req *http.Request) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
}

func SetStatusErrorf(ctx context.Context, format string, args ...interface{}) {
	if span := trace.SpanFromContext(ctx); span != nil {
		span.SetStatus(codes.Error, fmt.Sprintf(format, args...))
	}
}

type Span struct {
	trace.Span

	tracerProvider *TracerProvider
}

func (s Span) TracerProvider() trace.TracerProvider {
	return s.tracerProvider
}

type TracerProvider struct {
	trace.TracerProvider

	tracer *Tracer
}

func (t TracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	if name == opentelemetry.TracerName {
		return t.tracer
	}

	return t.TracerProvider.Tracer(name, options...)
}

type Tracer struct {
	trace.Tracer

	safeQueryParams         []string
	capturedRequestHeaders  []string
	capturedResponseHeaders []string
}

func NewTracer(tracer trace.Tracer, capturedRequestHeaders, capturedResponseHeaders, safeQueryParams []string) *Tracer {
	return &Tracer{
		Tracer:                  tracer,
		safeQueryParams:         safeQueryParams,
		capturedRequestHeaders:  capturedRequestHeaders,
		capturedResponseHeaders: capturedResponseHeaders,
	}
}

func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t == nil {
		return ctx, nil
	}

	spanCtx, span := t.Tracer.Start(ctx, spanName, opts...)

	wrappedSpan := &Span{Span: span, tracerProvider: &TracerProvider{tracer: t}}

	return trace.ContextWithSpan(spanCtx, wrappedSpan), wrappedSpan
}

func (t *Tracer) CaptureClientRequest(span trace.Span, r *http.Request) {
	if t == nil || span == nil || r == nil {
		return
	}

	span.SetAttributes(semconv.HTTPRequestMethodKey.String(r.Method))
	span.SetAttributes(semconv.NetworkProtocolVersion(proto(r.Proto)))

	sURL := t.safeURL(r.URL)
	span.SetAttributes(semconv.URLFull(sURL.String()))
	span.SetAttributes(semconv.URLScheme(sURL.Scheme))
	span.SetAttributes(semconv.UserAgentOriginal(r.UserAgent()))

	host, port, err := net.SplitHostPort(sURL.Host)
	if err != nil {
		span.SetAttributes(semconv.NetworkPeerAddress(host))
		span.SetAttributes(semconv.ServerAddress(sURL.Host))
		switch sURL.Scheme {
		case "http":
			span.SetAttributes(semconv.NetworkPeerPort(80))
			span.SetAttributes(semconv.ServerPort(80))
		case "https":
			span.SetAttributes(semconv.NetworkPeerPort(443))
			span.SetAttributes(semconv.ServerPort(443))
		}
	} else {
		span.SetAttributes(semconv.NetworkPeerAddress(host))
		intPort, _ := strconv.Atoi(port)
		span.SetAttributes(semconv.NetworkPeerPort(intPort))
		span.SetAttributes(semconv.ServerAddress(host))
		span.SetAttributes(semconv.ServerPort(intPort))
	}

	for _, header := range t.capturedRequestHeaders {
		// User-agent is already part of the semantic convention as a recommended attribute.
		if strings.EqualFold(header, "User-Agent") {
			continue
		}

		if value := r.Header[header]; value != nil {
			span.SetAttributes(attribute.StringSlice(fmt.Sprintf("http.request.header.%s", strings.ToLower(header)), value))
		}
	}
}

func (t *Tracer) CaptureServerRequest(span trace.Span, r *http.Request) {
	if t == nil || span == nil || r == nil {
		return
	}

	span.SetAttributes(semconv.HTTPRequestMethodKey.String(r.Method))
	span.SetAttributes(semconv.NetworkProtocolVersion(proto(r.Proto)))

	sURL := t.safeURL(r.URL)
	span.SetAttributes(semconv.HTTPRequestBodySize(int(r.ContentLength)))
	span.SetAttributes(semconv.URLPath(sURL.Path))
	span.SetAttributes(semconv.URLQuery(sURL.RawQuery))
	span.SetAttributes(semconv.URLScheme(r.Header.Get("X-Forwarded-Proto")))
	span.SetAttributes(semconv.UserAgentOriginal(r.UserAgent()))
	span.SetAttributes(semconv.ServerAddress(r.Host))

	host, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		span.SetAttributes(semconv.ClientAddress(r.RemoteAddr))
		span.SetAttributes(semconv.NetworkPeerAddress(r.Host))
	} else {
		span.SetAttributes(semconv.NetworkPeerAddress(host))
		span.SetAttributes(semconv.ClientAddress(host))
		intPort, _ := strconv.Atoi(port)
		span.SetAttributes(semconv.ClientPort(intPort))
		span.SetAttributes(semconv.NetworkPeerPort(intPort))
	}

	for _, header := range t.capturedRequestHeaders {
		if strings.EqualFold(header, "User-Agent") {
			continue
		}

		if value := r.Header[header]; value != nil {
			span.SetAttributes(attribute.StringSlice(fmt.Sprintf("http.request.header.%s", strings.ToLower(header)), value))
		}
	}
}

func (t *Tracer) CaptureResponse(span trace.Span, responseHeaders http.Header, code int, spanKind trace.SpanKind) {
	if t == nil || span == nil {
		return
	}

	var status codes.Code
	var desc string
	switch spanKind {
	case trace.SpanKindServer:
		status, desc = serverStatus(code)
	case trace.SpanKindClient:
		status, desc = clientStatus(code)
	default:
		status, desc = defaultStatus(code)
	}
	span.SetStatus(status, desc)
	if code > 0 {
		span.SetAttributes(semconv.HTTPResponseStatusCode(code))
	}

	for _, header := range t.capturedResponseHeaders {
		if value := responseHeaders[header]; value != nil {
			span.SetAttributes(attribute.StringSlice(fmt.Sprintf("http.response.header.%s", strings.ToLower(header)), value))
		}
	}
}

func (t *Tracer) safeURL(originalURL *url.URL) *url.URL {
	if originalURL == nil {
		return nil
	}

	redactedURL := *originalURL

	// Redact password if exists.
	if redactedURL.User != nil {
		redactedURL.User = url.UserPassword("REDACTED", "REDACTED")
	}

	query := redactedURL.Query()
	for k := range query {
		if slices.Contains(t.safeQueryParams, k) {
			continue
		}

		query.Set(k, "REDACTED")
	}
	redactedURL.RawQuery = query.Encode()

	return &redactedURL
}

func proto(proto string) string {
	switch proto {
	case "HTTP/1.0":
		return "1.0"
	case "HTTP/1.1":
		return "1.1"
	case "HTTP/2":
		return "2"
	case "HTTP/3":
		return "3"
	default:
		return proto
	}
}

func serverStatus(code int) (codes.Code, string) {
	if code < 100 || code >= 600 {
		return codes.Error, fmt.Sprintf("Invalid HTTP status code %d", code)
	}
	if code >= 500 {
		return codes.Error, ""
	}
	return codes.Unset, ""
}

func clientStatus(code int) (codes.Code, string) {
	if code < 100 || code >= 600 {
		return codes.Error, fmt.Sprintf("Invalid HTTP status code %d", code)
	}
	if code >= 400 {
		return codes.Error, ""
	}
	return codes.Unset, ""
}

func defaultStatus(code int) (codes.Code, string) {
	if code < 100 || code >= 600 {
		return codes.Error, fmt.Sprintf("Invalid HTTP status code %d", code)
	}
	if code >= 500 {
		return codes.Error, ""
	}
	return codes.Unset, ""
}
