package signals

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

func TestNotifyContext_CancelsOnSignal(t *testing.T) {
	ctx, cancel := NotifyContext(context.Background())
	defer cancel()

	// Simulate a SIGINT to ourselves.
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("kill: %v", err)
	}

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("expected Canceled, got %v", ctx.Err())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ctx did not cancel on SIGINT within 2s")
	}
}

func TestNotifyContext_ParentCancelPropagates(t *testing.T) {
	parent, pc := context.WithCancel(context.Background())
	ctx, cancel := NotifyContext(parent)
	defer cancel()

	pc() // cancel parent

	select {
	case <-ctx.Done():
		// ok
	case <-time.After(time.Second):
		t.Fatal("parent cancel should propagate")
	}
}

func TestWait_ReturnsOnSignal(t *testing.T) {
	type result struct{}
	resultCh := make(chan result, 1)

	go func() {
		_ = Wait(context.Background())
		resultCh <- result{}
	}()

	// Give the goroutine time to register the handler.
	time.Sleep(50 * time.Millisecond)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	select {
	case <-resultCh:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return on SIGTERM within 2s")
	}
}

func TestWait_ReturnsNilOnCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sig := Wait(ctx)
		if sig != nil {
			t.Errorf("expected nil on ctx cancel, got %v", sig)
		}
		close(done)
	}()
	cancel()

	select {
	case <-done:
		// ok
	case <-time.After(time.Second):
		t.Fatal("Wait did not return on ctx cancel")
	}
}
