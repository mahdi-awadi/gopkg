package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), DefaultPolicy(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	p := Policy{MaxAttempts: 5, InitialDelay: time.Millisecond, Multiplier: 1, JitterFraction: 0}
	err := Do(context.Background(), p, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_GivesUpAfterMaxAttempts(t *testing.T) {
	calls := 0
	p := Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, Multiplier: 1}
	err := Do(context.Background(), p, func() error {
		calls++
		return errors.New("never succeeds")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_PermanentErrorStopsRetry(t *testing.T) {
	calls := 0
	p := Policy{MaxAttempts: 10, InitialDelay: time.Millisecond, Multiplier: 1}
	err := Do(context.Background(), p, func() error {
		calls++
		return Permanent(errors.New("auth failed"))
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPermanent) {
		t.Fatalf("expected ErrPermanent, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call (permanent), got %d", calls)
	}
}

func TestDo_CtxCancellationAborts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	p := Policy{MaxAttempts: 10, InitialDelay: 10 * time.Millisecond, Multiplier: 1}
	err := Do(ctx, p, func() error { return errors.New("x") })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDo_MaxAttemptsClampedToMin1(t *testing.T) {
	calls := 0
	p := Policy{MaxAttempts: 0, InitialDelay: time.Millisecond}
	_ = Do(context.Background(), p, func() error {
		calls++
		return errors.New("x")
	})
	if calls != 1 {
		t.Fatalf("expected at least 1 call, got %d", calls)
	}
}

func TestPermanent_NilReturnsNil(t *testing.T) {
	if Permanent(nil) != nil {
		t.Fatal("Permanent(nil) should return nil")
	}
}
