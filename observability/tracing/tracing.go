// Package tracing provides OpenTelemetry tracer initialization with an
// OTLP gRPC exporter, plus a handful of helpers for span/context plumbing.
//
// Designed to be called once from main(), returns a shutdown func you
// defer on exit. Hardwired to W3C Trace Context + Baggage propagation.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds tracer configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	// OTLPEndpoint is the OTLP gRPC collector address (e.g. "otel-collector:4317",
	// "jaeger:4317"). Required unless Enabled=false.
	OTLPEndpoint string
	// SampleRate is the TraceIDRatioBased sampler rate in [0.0, 1.0].
	// 1.0 = sample everything, 0.0 = sample nothing.
	SampleRate float64
	// Enabled gates the whole init; when false, Init returns a no-op shutdown.
	Enabled bool
}

// Init installs a global tracer provider with the given config and returns
// a shutdown function. Call shutdown on process exit to flush pending spans.
//
// If cfg.Enabled is false, returns a no-op shutdown (tracing is not started).
func Init(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("tracing: ServiceName is required")
	}
	if cfg.OTLPEndpoint == "" {
		return nil, fmt.Errorf("tracing: OTLPEndpoint is required")
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 1.0
	}

	conn, err := grpc.DialContext(ctx, cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: gRPC dial %s: %w", cfg.OTLPEndpoint, err)
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(otlptracegrpc.WithGRPCConn(conn)))
	if err != nil {
		return nil, fmt.Errorf("tracing: create exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// TraceID extracts the current span's trace ID (hex string), or "" if unset.
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanID extracts the current span's span ID (hex string), or "" if unset.
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// IsSampled reports whether the current span is valid and sampled.
func IsSampled(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().IsValid() && span.SpanContext().IsSampled()
}
