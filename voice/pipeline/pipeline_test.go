package pipeline

import (
	"testing"
	"time"
)

func TestNew_ZeroOptionsAppliesDefaults(t *testing.T) {
	p, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.opts.ToolConcurrency != 1 {
		t.Errorf("default ToolConcurrency=%d, want 1", p.opts.ToolConcurrency)
	}
	if p.opts.HoldFillerDelay != 2*time.Second {
		t.Errorf("default HoldFillerDelay=%v, want 2s", p.opts.HoldFillerDelay)
	}
	if p.opts.Filler == nil {
		t.Error("default Filler must be non-nil (SilentFiller)")
	}
	if p.opts.Observer == nil {
		t.Error("default Observer must be non-nil (NoopObserver)")
	}
	if p.opts.Logger == nil {
		t.Error("default Logger must be non-nil (NoopLogger)")
	}
	if p.opts.SessionIDFunc == nil {
		t.Error("default SessionIDFunc must be non-nil")
	}
	// ID fn should produce a non-empty unique-ish string
	a, b := p.opts.SessionIDFunc(), p.opts.SessionIDFunc()
	if a == "" || a == b {
		t.Errorf("SessionIDFunc returned %q / %q", a, b)
	}
}

func TestNew_NegativeToolConcurrencyRejected(t *testing.T) {
	_, err := New(Options{ToolConcurrency: -1})
	if err == nil {
		t.Error("expected error for negative ToolConcurrency")
	}
}

func TestNew_NegativeHoldDelayRejected(t *testing.T) {
	_, err := New(Options{HoldFillerDelay: -1 * time.Second})
	if err == nil {
		t.Error("expected error for negative HoldFillerDelay")
	}
}

func TestNew_OverridesHonored(t *testing.T) {
	called := false
	fn := func() string { called = true; return "custom-id" }
	p, err := New(Options{
		ToolConcurrency: 4,
		HoldFillerDelay: 500 * time.Millisecond,
		SessionIDFunc:   fn,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.opts.ToolConcurrency != 4 {
		t.Errorf("ToolConcurrency=%d, want 4", p.opts.ToolConcurrency)
	}
	if p.opts.HoldFillerDelay != 500*time.Millisecond {
		t.Errorf("HoldFillerDelay=%v, want 500ms", p.opts.HoldFillerDelay)
	}
	if got := p.opts.SessionIDFunc(); got != "custom-id" || !called {
		t.Errorf("SessionIDFunc overridden but returned %q", got)
	}
}
