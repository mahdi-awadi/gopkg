package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func testCfg() Config {
	c := DefaultConfig()
	c.FailureThreshold = 3
	c.SuccessThreshold = 1
	c.OpenTimeout = 20 * time.Millisecond
	c.MaxOpenTimeout = 200 * time.Millisecond
	c.BackoffMultiplier = 2.0
	c.HalfOpenMaxRequests = 1
	return c
}

func TestClosedAllowsAllCalls(t *testing.T) {
	b := New("t", testCfg(), nil)
	for i := 0; i < 10; i++ {
		if err := b.Allow(); err != nil {
			t.Fatalf("closed should allow, got %v", err)
		}
	}
}

func TestOpensAfterThresholdFailures(t *testing.T) {
	b := New("t", testCfg(), nil)
	for i := 0; i < 3; i++ {
		b.RecordFailure("timeout")
	}
	if b.State() != StateOpen {
		t.Errorf("state=%s after 3 timeouts, want open", b.State())
	}
	if !errors.Is(b.Allow(), ErrCircuitOpen) {
		t.Errorf("Allow while open should return ErrCircuitOpen")
	}
}

func TestNonRetryableDoesNotTrip(t *testing.T) {
	b := New("t", testCfg(), nil)
	for i := 0; i < 10; i++ {
		b.RecordFailure("validation_error")
	}
	if b.State() != StateClosed {
		t.Errorf("validation_error must not trip breaker, state=%s", b.State())
	}
}

func TestHalfOpenAllowsProbeAndClosesOnSuccess(t *testing.T) {
	b := New("t", testCfg(), nil)
	for i := 0; i < 3; i++ {
		b.RecordFailure("timeout")
	}
	time.Sleep(30 * time.Millisecond) // past OpenTimeout
	// Allow should transition to half-open.
	if err := b.Allow(); err != nil {
		t.Fatalf("half-open probe should be allowed, got %v", err)
	}
	if b.State() != StateHalfOpen {
		t.Errorf("expected half-open, got %s", b.State())
	}
	// A second concurrent probe is blocked by HalfOpenMaxRequests=1.
	if !errors.Is(b.Allow(), ErrCircuitOpen) {
		t.Errorf("second probe in half-open should be rejected")
	}
	b.RecordSuccess()
	if b.State() != StateClosed {
		t.Errorf("success in half-open should close, got %s", b.State())
	}
}

func TestHalfOpenFailureReopensWithBackoff(t *testing.T) {
	b := New("t", testCfg(), nil)
	for i := 0; i < 3; i++ {
		b.RecordFailure("timeout")
	}
	before := b.Stats().CurrentOpenTimeout
	time.Sleep(30 * time.Millisecond)
	_ = b.Allow() // → half-open
	b.RecordFailure("timeout")
	if b.State() != StateOpen {
		t.Errorf("failed probe should reopen, got %s", b.State())
	}
	after := b.Stats().CurrentOpenTimeout
	if after <= before {
		t.Errorf("backoff should grow: before=%v after=%v", before, after)
	}
}

func TestBackoffCapsAtMaxOpenTimeout(t *testing.T) {
	cfg := testCfg()
	cfg.MaxOpenTimeout = 50 * time.Millisecond
	b := New("t", cfg, nil)

	// Drive the breaker into many reopens.
	for iter := 0; iter < 10; iter++ {
		for i := 0; i < 3; i++ {
			b.RecordFailure("timeout")
		}
		time.Sleep(b.Stats().CurrentOpenTimeout + 5*time.Millisecond)
		_ = b.Allow()
		b.RecordFailure("timeout")
	}
	if got := b.Stats().CurrentOpenTimeout; got > cfg.MaxOpenTimeout {
		t.Errorf("backoff=%v exceeds cap %v", got, cfg.MaxOpenTimeout)
	}
}

func TestReset(t *testing.T) {
	b := New("t", testCfg(), nil)
	for i := 0; i < 3; i++ {
		b.RecordFailure("timeout")
	}
	b.Reset()
	if b.State() != StateClosed {
		t.Errorf("Reset should close, state=%s", b.State())
	}
	if b.Failures() != 0 {
		t.Errorf("Reset should clear failure counter")
	}
}

func TestExecute_UsesDefaultClassify(t *testing.T) {
	b := New("t", testCfg(), nil)
	err := b.Execute(context.Background(), func(context.Context) error {
		return errors.New("operation timeout after 30s")
	}, nil)
	if err == nil {
		t.Fatal("expected error pass-through from fn")
	}
	if b.Failures() != 1 {
		t.Errorf("failure should have counted, got %d", b.Failures())
	}
}

func TestExecute_RejectsWhenOpen(t *testing.T) {
	b := New("t", testCfg(), nil)
	for i := 0; i < 3; i++ {
		b.RecordFailure("timeout")
	}
	called := false
	err := b.Execute(context.Background(), func(context.Context) error {
		called = true
		return nil
	}, nil)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
	if called {
		t.Errorf("fn should not run when open")
	}
}

func TestOnStateChangeCallback(t *testing.T) {
	b := New("t", testCfg(), nil)
	got := make(chan [3]string, 4)
	b.SetOnStateChange(func(name string, from, to State) {
		got <- [3]string{name, from.String(), to.String()}
	})
	for i := 0; i < 3; i++ {
		b.RecordFailure("timeout")
	}
	select {
	case ev := <-got:
		if ev[0] != "t" || ev[1] != "closed" || ev[2] != "open" {
			t.Errorf("unexpected event: %v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no state-change event fired")
	}
}

func TestDefaultClassify(t *testing.T) {
	cases := map[string]string{
		"":                         "unknown",
		"context deadline exceeded timeout": "timeout",
		"dial tcp: connection refused":       "connection_refused",
		"read: connection reset by peer":     "connection_reset",
		"dial tcp: no such host":             "dns_error",
		"429 rate limit exceeded":            "rate_limited",
		"upstream 502 Bad Gateway":           "service_unavailable",
		"HTTP 500 internal":                  "server_error",
		"401 Unauthorized":                   "auth_error",
		"403 Forbidden":                      "forbidden",
		"invalid request 400":                "validation_error",
		"404 not found":                      "not_found",
		"something weird":                    "unknown",
	}
	for msg, want := range cases {
		var err error
		if msg != "" {
			err = errors.New(msg)
		} else {
			err = errors.New("x")
			want = "unknown"
		}
		if got := DefaultClassify(err); got != want {
			t.Errorf("Classify(%q)=%q, want %q", msg, got, want)
		}
	}
	if DefaultClassify(nil) != "" {
		t.Errorf("Classify(nil) should be empty")
	}
}

func TestStateString(t *testing.T) {
	if StateClosed.String() != "closed" || StateOpen.String() != "open" ||
		StateHalfOpen.String() != "half_open" || State(99).String() != "unknown" {
		t.Errorf("State.String() returned unexpected labels")
	}
}
