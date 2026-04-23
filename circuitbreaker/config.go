package circuitbreaker

import "time"

// Config holds tuning knobs for a Breaker.
//
// The zero value is NOT valid — every breaker needs at least
// FailureThreshold > 0 and SuccessThreshold > 0. Use DefaultConfig()
// and adjust from there.
type Config struct {
	// FailureThreshold is the consecutive-failure count that trips the
	// breaker from closed → open.
	FailureThreshold uint32

	// SuccessThreshold is the consecutive-success count needed to
	// close the breaker from half-open → closed.
	SuccessThreshold uint32

	// OpenTimeout is the minimum time to wait in open state before
	// allowing a probe through. This is the *base* value; the actual
	// timeout grows per consecutive open (see BackoffMultiplier).
	OpenTimeout time.Duration

	// MaxOpenTimeout caps the exponential backoff. Zero means "no cap".
	MaxOpenTimeout time.Duration

	// BackoffMultiplier multiplies the current open-timeout on every
	// failed half-open probe. 2.0 doubles each time; 1.0 disables
	// growth; 0 also disables growth.
	BackoffMultiplier float64

	// HalfOpenMaxRequests is the concurrent-probe cap in half-open
	// state. Typically 1 (classic), higher for throughput-sensitive
	// cases where parallel probes are acceptable.
	HalfOpenMaxRequests uint32

	// RetryableErrors is the allow-list of bucket strings that count
	// against the failure budget. An error whose classified bucket is
	// in this list increments the failure counter.
	RetryableErrors []string

	// NonRetryableErrors is the explicit deny-list — buckets here do
	// NOT count against the failure budget, even if they're also in
	// RetryableErrors. Typical examples: "validation_error",
	// "not_found", "auth_error" — these are caller-side problems, not
	// downstream-health signals.
	NonRetryableErrors []string
}

// DefaultConfig returns a sensible baseline for HTTP/RPC clients:
// 5 failures trip the breaker, a single success closes it, base
// open-timeout is 30 s, capped at 1 h, doubling per failed probe.
func DefaultConfig() Config {
	return Config{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		OpenTimeout:         30 * time.Second,
		MaxOpenTimeout:      1 * time.Hour,
		BackoffMultiplier:   2.0,
		HalfOpenMaxRequests: 1,
		RetryableErrors: []string{
			"timeout",
			"connection_refused",
			"connection_reset",
			"dns_error",
			"rate_limited",
			"service_unavailable",
			"server_error",
		},
		NonRetryableErrors: []string{
			"auth_error",
			"forbidden",
			"validation_error",
			"not_found",
		},
	}
}

// IsRetryableError reports whether a given classified bucket counts
// toward the failure budget.
//
// Precedence: explicit NonRetryable > explicit Retryable > default (true).
// The default-true bias is deliberate: unknown failures count, so
// previously-unseen downstream trouble trips the breaker rather than
// slipping past unnoticed.
func (c *Config) IsRetryableError(bucket string) bool {
	for _, e := range c.NonRetryableErrors {
		if e == bucket {
			return false
		}
	}
	for _, e := range c.RetryableErrors {
		if e == bucket {
			return true
		}
	}
	return true
}
