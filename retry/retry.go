// Package retry provides context-aware exponential backoff with jitter.
//
// The exported API is deliberately small:
//
//	err := retry.Do(ctx, policy, func() error { return callExternalService() })
//
// For advanced control (per-attempt logging, early abort on specific errors),
// consumers can wrap Do with their own closure.
//
// Zero third-party deps.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// Policy controls how retries behave.
//
// Zero value is invalid; use DefaultPolicy as a starting point.
type Policy struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	// Values < 1 are clamped to 1.
	MaxAttempts int

	// InitialDelay is the delay before the second attempt.
	InitialDelay time.Duration

	// MaxDelay caps the computed backoff. Zero means no cap.
	MaxDelay time.Duration

	// Multiplier is applied to the delay on each attempt (e.g. 2.0 doubles).
	Multiplier float64

	// JitterFraction is the fraction [0.0, 1.0] of delay added as uniform
	// random jitter. 0.0 = deterministic, 0.25 = ±25%.
	JitterFraction float64
}

// DefaultPolicy returns a reasonable default: 5 attempts, 100ms initial delay,
// 5s cap, 2x multiplier, 25% jitter.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:    5,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       5 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.25,
	}
}

// Permanent wraps an error to signal that Do should NOT retry it.
// errors.Is(err, ErrPermanent) reports whether the error is wrapped.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// ErrPermanent is the sentinel returned by errors.Is on a permanent error.
var ErrPermanent = errors.New("retry: permanent error")

type permanentError struct{ err error }

func (p *permanentError) Error() string { return p.err.Error() }
func (p *permanentError) Unwrap() error { return p.err }
func (p *permanentError) Is(target error) bool {
	return target == ErrPermanent
}

// Do runs fn up to policy.MaxAttempts times, sleeping between attempts
// with exponential backoff + jitter. Stops early on:
//   - nil return (success)
//   - Permanent(err) wrapped error
//   - ctx cancellation
func Do(ctx context.Context, policy Policy, fn func() error) error {
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}
	if policy.Multiplier < 1 {
		policy.Multiplier = 1
	}
	if policy.JitterFraction < 0 {
		policy.JitterFraction = 0
	}
	if policy.JitterFraction > 1 {
		policy.JitterFraction = 1
	}

	var lastErr error
	delay := policy.InitialDelay
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if errors.Is(lastErr, ErrPermanent) {
			return lastErr
		}
		if attempt == policy.MaxAttempts {
			break
		}

		sleep := applyJitter(delay, policy.JitterFraction)
		if policy.MaxDelay > 0 && sleep > policy.MaxDelay {
			sleep = policy.MaxDelay
		}

		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		// advance delay for next attempt
		next := time.Duration(float64(delay) * policy.Multiplier)
		if policy.MaxDelay > 0 && next > policy.MaxDelay {
			next = policy.MaxDelay
		}
		delay = next
	}

	return fmt.Errorf("retry: gave up after %d attempts: %w", policy.MaxAttempts, lastErr)
}

func applyJitter(d time.Duration, frac float64) time.Duration {
	if frac <= 0 {
		return d
	}
	// Uniform jitter in [d*(1-frac), d*(1+frac)]
	jitter := (rand.Float64()*2 - 1) * frac
	out := time.Duration(float64(d) * (1 + jitter))
	if out < 0 {
		out = 0
	}
	return out
}
