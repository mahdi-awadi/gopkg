package workerpool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubmit_AllTasksRun(t *testing.T) {
	p := New(5)
	var count int64
	for i := 0; i < 100; i++ {
		p.Submit(func() { atomic.AddInt64(&count, 1) })
	}
	p.Wait()
	if atomic.LoadInt64(&count) != 100 {
		t.Fatalf("expected 100, got %d", count)
	}
}

func TestWait_Idempotent(t *testing.T) {
	p := New(2)
	p.Submit(func() {})
	p.Wait()
	p.Wait() // should not panic
}

func TestSubmit_AfterWaitPanics(t *testing.T) {
	p := New(2)
	p.Wait()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	p.Submit(func() {})
}

func TestSubmitCtx_CancelAborts(t *testing.T) {
	p := New(1)
	defer p.Wait()

	// occupy the single worker
	started := make(chan struct{})
	hold := make(chan struct{})
	p.Submit(func() {
		close(started)
		<-hold
	})
	<-started

	// Any further submit should block on the unbuffered channel.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := p.SubmitCtx(ctx, func() {})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	close(hold)
}

func TestSubmitCtx_AfterCloseReturnsErrClosed(t *testing.T) {
	p := New(1)
	p.Wait()
	err := p.SubmitCtx(context.Background(), func() {})
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestPanickingTask_DoesNotKillWorker(t *testing.T) {
	p := New(2)
	var ok int64
	p.Submit(func() { panic("boom") })
	p.Submit(func() { atomic.StoreInt64(&ok, 1) })
	p.Wait()
	if ok != 1 {
		t.Fatal("second task should have run despite prior panic")
	}
}
