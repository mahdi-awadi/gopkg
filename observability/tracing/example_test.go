package tracing_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/observability/tracing"
)

func Example() {
	shutdown, _ := tracing.Init(context.Background(), tracing.Config{
		ServiceName:    "demo",
		ServiceVersion: "v1.0.0",
		Environment:    "development",
		OTLPEndpoint:   "otel-collector:4317",
		SampleRate:     1.0,
		Enabled:        false, // set true in production
	})
	defer shutdown(context.Background())

	ctx, span := tracing.Tracer("demo").Start(context.Background(), "work")
	defer span.End()

	fmt.Println("trace_id:", tracing.TraceID(ctx)) // empty — tracing disabled
}
