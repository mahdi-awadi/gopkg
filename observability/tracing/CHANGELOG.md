# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Config{ServiceName, ServiceVersion, Environment, OTLPEndpoint, SampleRate, Enabled}`
- `Init(ctx, cfg)` installs global tracer provider; returns shutdown func
- `Tracer(name)` returns a named tracer from the global provider
- `TraceID(ctx)` / `SpanID(ctx)` / `IsSampled(ctx)` helpers
- Hardcoded W3C Trace Context + Baggage propagators
- Disabled config returns no-op shutdown (safe for dev/test)
- 5 tests

### Dependencies
- OpenTelemetry SDK + OTLP/gRPC exporter v1.29.0
- google.golang.org/grpc v1.65.0
