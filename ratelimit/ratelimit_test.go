package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAllow_BurstThenRefill(t *testing.T) {
	l := New(3, 10) // 3 burst, 10 tokens/sec
	// Manually control "now" to remove flakiness.
	start := time.Now()
	l.now = func() time.Time { return start }

	// 3 immediate successes
	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Fatalf("burst call %d should Allow", i)
		}
	}
	if l.Allow() {
		t.Fatal("4th call should be denied (burst exhausted)")
	}

	// advance 1 second → 10 tokens refilled (capped at 3)
	l.now = func() time.Time { return start.Add(1 * time.Second) }
	if !l.Allow() {
		t.Fatal("after 1s refill should Allow")
	}
}

func TestAllow_CapsAtCapacity(t *testing.T) {
	l := New(5, 100)
	start := time.Now()
	l.now = func() time.Time { return start }

	// Move time forward a LOT — refill should cap at 5.
	l.now = func() time.Time { return start.Add(1 * time.Hour) }
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatalf("should allow up to capacity; failed at %d", i)
		}
	}
	if l.Allow() {
		t.Fatal("6th call should be denied (capacity cap)")
	}
}

func TestWait_CtxCancel(t *testing.T) {
	l := New(1, 0.1) // very slow refill
	// Consume the single token.
	if !l.Allow() {
		t.Fatal("initial Allow should pass")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := l.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestClose_ReturnsErrClosed(t *testing.T) {
	l := New(1, 0.01)
	_ = l.Allow() // drain
	l.Close()
	// Try Wait — should return ErrClosed on the next refill-check.
	err := l.Wait(context.Background())
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestNewEvery(t *testing.T) {
	l := NewEvery(10, time.Second)
	if !l.Allow() {
		t.Fatal("NewEvery should start with capacity tokens")
	}
}

func TestNew_PanicsOnBadArgs(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for capacity=0")
		}
	}()
	_ = New(0, 1)
}
