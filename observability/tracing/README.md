# observability/tracing

OpenTelemetry tracer initialization with an OTLP gRPC exporter. Defaults to W3C Trace Context + Baggage propagation.

```
go get github.com/mahdi-awadi/gopkg/observability/tracing@latest
```

## Quickstart

```go
import (
    "context"
    "github.com/mahdi-awadi/gopkg/observability/tracing"
)

shutdown, err := tracing.Init(context.Background(), tracing.Config{
    ServiceName:    "my-service",
    ServiceVersion: "v1.2.3",
    Environment:    "production",
    OTLPEndpoint:   "otel-collector:4317",
    SampleRate:     1.0,
    Enabled:        true,
})
if err != nil { /* handle */ }
defer shutdown(context.Background())

tracer := tracing.Tracer("my-service")
ctx, span := tracer.Start(ctx, "HandleRequest")
defer span.End()

log.Println("trace_id:", tracing.TraceID(ctx))
```

## Behavior

- `Enabled=false` → returns a no-op shutdown; no TracerProvider installed
- Sampler: `TraceIDRatioBased(SampleRate)` (default 1.0 = sample everything)
- Propagator: W3C Trace Context + Baggage
- Exporter: OTLP gRPC with `insecure` credentials (OK inside a VPC; put a sidecar in front otherwise)

## License

[MIT](../../LICENSE)
