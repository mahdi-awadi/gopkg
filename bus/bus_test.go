package bus

import (
	"errors"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{name: "empty", cfg: Config{}, wantErr: nil},
		{name: "service-only", cfg: Config{ServiceName: "x"}, wantErr: nil},
		{name: "full", cfg: Config{Type: "nats", URL: "nats://x:4222", ServiceName: "x"}, wantErr: nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Config has no Validate method in v0.1.0; shape-check the fields are stable.
			_ = c.cfg.Type
			_ = c.cfg.URL
			_ = c.cfg.User
			_ = c.cfg.Password
			_ = c.cfg.ServiceName
		})
	}
	_ = errors.New
}

func TestApplyPublishOptions_Empty(t *testing.T) {
	cfg := ApplyPublishOptions(nil)
	if cfg.OrderingKey != "" {
		t.Fatalf("expected empty OrderingKey, got %q", cfg.OrderingKey)
	}
	if len(cfg.Headers) != 0 {
		t.Fatalf("expected empty Headers, got %v", cfg.Headers)
	}
}

func TestWithOrderingKey(t *testing.T) {
	cfg := ApplyPublishOptions([]PublishOption{WithOrderingKey("k1")})
	if cfg.OrderingKey != "k1" {
		t.Fatalf("got %q, want k1", cfg.OrderingKey)
	}
}

func TestWithHeaders_MergesMultiple(t *testing.T) {
	cfg := ApplyPublishOptions([]PublishOption{
		WithHeaders(map[string]string{"a": "1"}),
		WithHeaders(map[string]string{"b": "2"}),
	})
	if cfg.Headers["a"] != "1" || cfg.Headers["b"] != "2" {
		t.Fatalf("expected merged headers, got %v", cfg.Headers)
	}
}

func TestNoopLogger_DiscardsCalls(t *testing.T) {
	var l Logger = NoopLogger{}
	l.Info("ignored", map[string]any{"k": "v"})
	l.Error("also ignored", nil)
	// no panic, no assertion — just that NoopLogger satisfies Logger
}

func TestWrapZap_NilReturnsNoop(t *testing.T) {
	l := WrapZap(nil)
	if _, ok := l.(NoopLogger); !ok {
		t.Fatalf("expected NoopLogger, got %T", l)
	}
}
