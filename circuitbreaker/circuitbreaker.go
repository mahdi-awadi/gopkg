// Package circuitbreaker implements the three-state circuit-breaker
// pattern (closed → open → half-open → closed) with exponential
// backoff on repeated failures.
//
// Calls that would breach a downstream dependency are rejected fast
// with ErrCircuitOpen once a failure threshold is crossed, sparing the
// downstream system and bounding caller latency. A single successful
// half-open probe closes the circuit; a failed probe reopens it with
// an expanded timeout (capped at MaxOpenTimeout).
//
// The package is classifier-agnostic: callers either call Record
// directly, or pass a ClassifyFunc into Execute that maps any error
// to a string bucket. Bucket strings are matched against
// Config.RetryableErrors / Config.NonRetryableErrors to decide whether
// a given error counts against the failure budget.
//
// Zero third-party dependencies. Safe for concurrent use.
package circuitbreaker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

// State is the circuit breaker's current state.
type State int

const (
	// StateClosed lets every call through.
	StateClosed State = iota
	// StateOpen rejects every call with ErrCircuitOpen.
	StateOpen
	// StateHalfOpen lets a small probe through to test whether the
	// downstream has recovered.
	StateHalfOpen
)

// String returns a stable lowercase label.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the breaker is rejecting traffic.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Logger is the minimum observability contract.
// The zero value (NoopLogger) is fine for no-op logging.
type Logger interface {
	Debug(msg string, fields map[string]any)
	Info(msg string, fields map[string]any)
}

// NoopLogger discards every log call.
type NoopLogger struct{}

// Debug discards the call.
func (NoopLogger) Debug(string, map[string]any) {}

// Info discards the call.
func (NoopLogger) Info(string, map[string]any) {}

// ClassifyFunc maps a non-nil error to a short bucket string (e.g.
// "timeout", "rate_limited"). The empty string means "don't count
// this error against the failure budget" (equivalent to
// NonRetryableErrors).
//
// DefaultClassify is a reasonable starting point; callers with richer
// error types should pass their own.
type ClassifyFunc func(err error) string

// Breaker protects a downstream dependency. Instances MUST be created
// via New; the zero value is not valid. Safe for concurrent use.
type Breaker struct {
	name   string
	config Config
	logger Logger

	mu               sync.RWMutex
	state            State
	failures         uint32
	successes        uint32
	halfOpenRequests uint32
	lastStateChange  time.Time
	lastFailure      time.Time

	consecutiveOpens   uint32
	currentOpenTimeout time.Duration

	onStateChange func(name string, from, to State)
}

// New constructs a Breaker.
// If logger is nil, NoopLogger{} is used.
func New(name string, cfg Config, logger Logger) *Breaker {
	if logger == nil {
		logger = NoopLogger{}
	}
	return &Breaker{
		name:               name,
		config:             cfg,
		logger:             logger,
		state:              StateClosed,
		lastStateChange:    time.Now(),
		currentOpenTimeout: cfg.OpenTimeout,
	}
}

// SetOnStateChange registers a callback invoked (in a new goroutine)
// whenever the breaker transitions between states. Pass nil to unset.
func (b *Breaker) SetOnStateChange(fn func(name string, from, to State)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onStateChange = fn
}

// Name returns the breaker's label.
func (b *Breaker) Name() string { return b.name }

// State returns the current state.
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Failures returns the current failure counter (resets on transitions).
func (b *Breaker) Failures() uint32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.failures
}

// Allow checks whether the next call may proceed. Returns nil if yes,
// ErrCircuitOpen if the breaker is rejecting traffic.
//
// Most callers should use Execute instead; Allow is exported for
// advanced cases where the caller must perform the protected work
// inline and report the outcome via RecordSuccess / RecordFailure.
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(b.lastStateChange) >= b.currentOpenTimeout {
			b.transitionTo(StateHalfOpen)
			b.halfOpenRequests = 1
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		if b.halfOpenRequests < b.config.HalfOpenMaxRequests {
			b.halfOpenRequests++
			return nil
		}
		return ErrCircuitOpen
	default:
		return nil
	}
}

// Execute runs fn under breaker protection. If the breaker is open,
// fn is not called and ErrCircuitOpen is returned. Otherwise fn runs,
// and its error (if any) is classified via `classify` and recorded.
//
// classify may be nil; DefaultClassify is used in that case.
func (b *Breaker) Execute(ctx context.Context, fn func(context.Context) error, classify ClassifyFunc) error {
	if err := b.Allow(); err != nil {
		return err
	}
	err := fn(ctx)
	if err != nil {
		if classify == nil {
			classify = DefaultClassify
		}
		b.RecordFailure(classify(err))
	} else {
		b.RecordSuccess()
	}
	return err
}

// RecordSuccess reports a successful protected call.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.failures = 0
	case StateHalfOpen:
		b.successes++
		if b.successes >= b.config.SuccessThreshold {
			b.transitionTo(StateClosed)
		}
	case StateOpen:
		// Shouldn't happen (Allow would have gated it), tolerate gracefully.
		b.successes++
	}
}

// RecordFailure reports a failed protected call with a pre-classified
// bucket string. Pass the empty string to record a failure that
// doesn't count against the failure budget.
func (b *Breaker) RecordFailure(errorBucket string) {
	if errorBucket == "" || !b.config.IsRetryableError(errorBucket) {
		b.logger.Debug("circuitbreaker: non-counting failure", map[string]any{
			"circuit": b.name,
			"bucket":  errorBucket,
		})
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastFailure = time.Now()

	switch b.state {
	case StateClosed:
		b.failures++
		if b.failures >= b.config.FailureThreshold {
			b.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		// A single failure in the probe returns us to open.
		b.transitionTo(StateOpen)
	case StateOpen:
		b.failures++
	}
}

// Reset forces the breaker back to StateClosed and resets its
// backoff. Useful for tests and for admin-initiated recovery.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionTo(StateClosed)
}

// Stats is a snapshot of the breaker's counters at a point in time.
type Stats struct {
	Name               string
	State              string
	Failures           uint32
	Successes          uint32
	ConsecutiveOpens   uint32
	CurrentOpenTimeout time.Duration
	LastStateChange    time.Time
	LastFailure        time.Time
}

// Stats returns a point-in-time snapshot. Safe for concurrent use.
func (b *Breaker) Stats() Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return Stats{
		Name:               b.name,
		State:              b.state.String(),
		Failures:           b.failures,
		Successes:          b.successes,
		ConsecutiveOpens:   b.consecutiveOpens,
		CurrentOpenTimeout: b.currentOpenTimeout,
		LastStateChange:    b.lastStateChange,
		LastFailure:        b.lastFailure,
	}
}

// transitionTo changes state; caller must hold b.mu.
func (b *Breaker) transitionTo(next State) {
	if b.state == next {
		return
	}
	prev := b.state
	b.state = next
	b.lastStateChange = time.Now()

	switch next {
	case StateClosed:
		b.failures = 0
		b.successes = 0
		b.halfOpenRequests = 0
		b.consecutiveOpens = 0
		b.currentOpenTimeout = b.config.OpenTimeout
	case StateOpen:
		b.successes = 0
		b.halfOpenRequests = 0
		if prev == StateHalfOpen {
			b.consecutiveOpens++
			if b.config.BackoffMultiplier > 0 {
				b.currentOpenTimeout = time.Duration(float64(b.currentOpenTimeout) * b.config.BackoffMultiplier)
				if b.config.MaxOpenTimeout > 0 && b.currentOpenTimeout > b.config.MaxOpenTimeout {
					b.currentOpenTimeout = b.config.MaxOpenTimeout
				}
			}
		}
	case StateHalfOpen:
		b.successes = 0
		b.halfOpenRequests = 0
	}

	b.logger.Info("circuitbreaker: state transition", map[string]any{
		"circuit":           b.name,
		"from":              prev.String(),
		"to":                next.String(),
		"open_timeout":      b.currentOpenTimeout,
		"consecutive_opens": b.consecutiveOpens,
	})

	if b.onStateChange != nil {
		go b.onStateChange(b.name, prev, next)
	}
}

// DefaultClassify is a best-effort classifier keyed off common
// error-message substrings. It's deliberately simple; callers with
// structured errors (errors.Is / typed sentinels) should pass a
// ClassifyFunc that returns buckets from the structure directly.
//
// Returns one of:
//
//	"" (when err is nil)
//	"timeout", "connection_refused", "connection_reset",
//	"dns_error", "rate_limited",
//	"service_unavailable" (HTTP 502/503), "server_error" (HTTP 500),
//	"auth_error" (401), "forbidden" (403),
//	"validation_error" (400), "not_found" (404),
//	"unknown" (fallback)
func DefaultClassify(err error) string {
	if err == nil {
		return ""
	}
	s := strings.ToLower(err.Error())
	switch {
	case strings.Contains(s, "timeout"):
		return "timeout"
	case strings.Contains(s, "connection refused"):
		return "connection_refused"
	case strings.Contains(s, "connection reset"):
		return "connection_reset"
	case strings.Contains(s, "no such host"):
		return "dns_error"
	case strings.Contains(s, "rate limit"):
		return "rate_limited"
	case strings.Contains(s, "503"), strings.Contains(s, "502"):
		return "service_unavailable"
	case strings.Contains(s, "500"):
		return "server_error"
	case strings.Contains(s, "401"), strings.Contains(s, "unauthorized"):
		return "auth_error"
	case strings.Contains(s, "403"), strings.Contains(s, "forbidden"):
		return "forbidden"
	case strings.Contains(s, "400"), strings.Contains(s, "validation"):
		return "validation_error"
	case strings.Contains(s, "404"), strings.Contains(s, "not found"):
		return "not_found"
	default:
		return "unknown"
	}
}
