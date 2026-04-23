package tracing

import (
	"context"
	"testing"
)

func TestInit_DisabledReturnsNoop(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown is nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestInit_RequiresServiceName(t *testing.T) {
	_, err := Init(context.Background(), Config{Enabled: true})
	if err == nil {
		t.Fatal("expected error for missing ServiceName")
	}
}

func TestInit_RequiresOTLPEndpoint(t *testing.T) {
	_, err := Init(context.Background(), Config{Enabled: true, ServiceName: "svc"})
	if err == nil {
		t.Fatal("expected error for missing OTLPEndpoint")
	}
}

func TestTraceIDAndSpanID_EmptyOnUnsetContext(t *testing.T) {
	if TraceID(context.Background()) != "" {
		t.Fatal("expected empty TraceID")
	}
	if SpanID(context.Background()) != "" {
		t.Fatal("expected empty SpanID")
	}
	if IsSampled(context.Background()) {
		t.Fatal("expected IsSampled=false on empty context")
	}
}

func TestTracer_NameableFromGlobal(t *testing.T) {
	// Without Init, Tracer() still returns a no-op tracer from OTel globals.
	tr := Tracer("test")
	if tr == nil {
		t.Fatal("expected non-nil tracer from global")
	}
}
