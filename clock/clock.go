// Package clock provides a tiny time-source abstraction for testability.
//
// Production code takes a Clock interface and receives Real. Tests pass
// a *Mock whose time advances only when Advance is called — eliminating
// flaky time.Sleep-based tests.
//
// Zero third-party deps.
package clock

import (
	"sync"
	"time"
)

// Clock is the minimum contract: current time + after-channel.
//
// Implementations MUST be safe for concurrent use.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// After returns a channel that receives the current time after d.
	// Implementations may return a buffered channel of size 1.
	After(d time.Duration) <-chan time.Time
	// Since is a convenience: time.Now().Sub(t), using the Clock's time.
	Since(t time.Time) time.Duration
}

// Real is the production Clock backed by time.Now / time.After.
type Real struct{}

// Now returns time.Now.
func (Real) Now() time.Time { return time.Now() }

// After returns time.After(d).
func (Real) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Since is time.Since.
func (Real) Since(t time.Time) time.Duration { return time.Since(t) }

// Mock is a controllable Clock for tests.
//
// Mock is safe for concurrent use. All timers fire when Advance moves
// the mock clock past their deadline.
type Mock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*waiter
}

type waiter struct {
	deadline time.Time
	ch       chan time.Time
}

// NewMock returns a Mock starting at the given time.
// Zero time uses time.Unix(0, 0).UTC() (i.e. the Unix epoch).
func NewMock(start time.Time) *Mock {
	if start.IsZero() {
		start = time.Unix(0, 0).UTC()
	}
	return &Mock{now: start}
}

// Now returns the mock's current time.
func (m *Mock) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.now
}

// Since is Now().Sub(t) on the mock clock.
func (m *Mock) Since(t time.Time) time.Duration {
	return m.Now().Sub(t)
}

// After returns a channel that fires when Advance moves the clock
// past the current time + d.
func (m *Mock) After(d time.Duration) <-chan time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan time.Time, 1)
	w := &waiter{deadline: m.now.Add(d), ch: ch}
	m.waiters = append(m.waiters, w)
	// If d <= 0, fire immediately.
	if !w.deadline.After(m.now) {
		w.ch <- m.now
		m.waiters = m.waiters[:len(m.waiters)-1]
	}
	return ch
}

// Advance moves the mock clock forward by d and fires any waiters
// whose deadlines are now due.
func (m *Mock) Advance(d time.Duration) {
	m.mu.Lock()
	m.now = m.now.Add(d)
	remaining := m.waiters[:0]
	var fire []*waiter
	for _, w := range m.waiters {
		if !w.deadline.After(m.now) {
			fire = append(fire, w)
		} else {
			remaining = append(remaining, w)
		}
	}
	m.waiters = remaining
	now := m.now
	m.mu.Unlock()

	for _, w := range fire {
		w.ch <- now
	}
}

// Set moves the mock clock to an absolute time, firing any waiters now due.
func (m *Mock) Set(t time.Time) {
	m.mu.Lock()
	m.now = t
	m.mu.Unlock()
	m.Advance(0) // fires any waiters already due
}
