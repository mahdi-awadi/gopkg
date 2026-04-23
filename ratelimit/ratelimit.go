// Package ratelimit provides a tiny thread-safe in-memory token bucket.
//
// Use per-request gating: `Allow()` for try-once, `Wait(ctx)` to block
// until a token is available.
//
// Zero third-party deps.
package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrClosed is returned by Wait when the Limiter is closed mid-wait.
var ErrClosed = errors.New("ratelimit: limiter closed")

// Limiter is a token-bucket limiter.
//
// Safe for concurrent use.
type Limiter struct {
	mu         sync.Mutex
	capacity   float64
	refillRate float64 // tokens per second
	tokens     float64
	lastRefill time.Time
	closed     bool
	now        func() time.Time
}

// New returns a Limiter that allows up to `capacity` operations in a burst
// and refills at `rate` tokens per second.
//
// Both must be positive.
func New(capacity, ratePerSecond float64) *Limiter {
	if capacity <= 0 || ratePerSecond <= 0 {
		panic("ratelimit: capacity and rate must be positive")
	}
	return &Limiter{
		capacity:   capacity,
		refillRate: ratePerSecond,
		tokens:     capacity, // start full
		lastRefill: time.Now(),
		now:        time.Now,
	}
}

// NewEvery is a convenience: rate = 1 op per `interval`.
func NewEvery(capacity int, interval time.Duration) *Limiter {
	return New(float64(capacity), 1.0/interval.Seconds())
}

// Allow reports whether a single token is available NOW; if so, consumes it.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available or ctx is cancelled.
// Returns ctx.Err() on cancel, ErrClosed if Close was called.
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			return ErrClosed
		}
		l.refill()
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		// How long until we'll have 1 token?
		deficit := 1 - l.tokens
		waitDur := time.Duration(deficit / l.refillRate * float64(time.Second))
		l.mu.Unlock()

		timer := time.NewTimer(waitDur)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Close marks the limiter closed; pending Wait calls return ErrClosed.
func (l *Limiter) Close() {
	l.mu.Lock()
	l.closed = true
	l.mu.Unlock()
}

// Tokens returns the current (fractional) token count — primarily for tests.
func (l *Limiter) Tokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	return l.tokens
}

func (l *Limiter) refill() {
	now := l.now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	l.tokens += elapsed * l.refillRate
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
	l.lastRefill = now
}
