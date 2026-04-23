// Package signals provides context helpers for graceful shutdown on
// OS signals (SIGINT, SIGTERM).
//
// Typical usage:
//
//	ctx := signals.NotifyContext(context.Background())
//	if err := server.Run(ctx); err != nil { ... }
//
// When SIGINT/SIGTERM arrives, ctx is cancelled — Run observes it and
// shuts down cleanly.
//
// Zero third-party deps.
package signals

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// DefaultSignals is the baseline set recognized by NotifyContext:
// SIGINT (Ctrl-C), SIGTERM (docker/k8s stop).
func DefaultSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}

// NotifyContext returns a derived context that is cancelled the first
// time one of DefaultSignals is received. Use NotifyContextFor for
// a custom signal set.
//
// The returned cancel func can be deferred; it also stops the internal
// signal listener.
func NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return NotifyContextFor(parent, DefaultSignals()...)
}

// NotifyContextFor is NotifyContext with an explicit signal list.
func NotifyContextFor(parent context.Context, sigs ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)

	go func() {
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(ch)
	}()

	return ctx, cancel
}

// Wait blocks until one of DefaultSignals is received or ctx is cancelled.
// Returns the received signal (or nil on ctx cancel).
func Wait(ctx context.Context) os.Signal {
	return WaitFor(ctx, DefaultSignals()...)
}

// WaitFor is Wait with an explicit signal list.
func WaitFor(ctx context.Context, sigs ...os.Signal) os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	defer signal.Stop(ch)

	select {
	case sig := <-ch:
		return sig
	case <-ctx.Done():
		return nil
	}
}
