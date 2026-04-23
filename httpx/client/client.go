// Package client provides a thin builder over *http.Client with sensible
// defaults for outbound HTTP: sane timeouts, connection pooling, optional
// retry on transient errors.
//
// Not an HTTP-request library (no BodyJSON sugar). Just a better
// constructor for the standard http.Client.
//
// Zero third-party deps.
package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Config configures New.
type Config struct {
	// Timeout is the total per-request deadline (includes dialing, TLS,
	// body read). Defaults to 30s.
	Timeout time.Duration
	// DialTimeout is the TCP connect timeout. Defaults to 5s.
	DialTimeout time.Duration
	// TLSHandshakeTimeout defaults to 5s.
	TLSHandshakeTimeout time.Duration
	// IdleConnTimeout defaults to 90s.
	IdleConnTimeout time.Duration
	// MaxIdleConns caps pooled idle connections. Default 100.
	MaxIdleConns int
	// MaxIdleConnsPerHost caps per-host. Default 10.
	MaxIdleConnsPerHost int
	// DisableKeepAlives turns off keep-alives if true.
	DisableKeepAlives bool
	// Retry configures automatic retry on transient errors.
	// Zero value = no retries.
	Retry RetryConfig
	// Transport overrides the default http.Transport. If set, the above
	// *Timeout / *Conn fields are IGNORED.
	Transport http.RoundTripper
}

// RetryConfig controls retry behavior.
type RetryConfig struct {
	// MaxAttempts — total attempts (including the first). 0 or 1 = no retry.
	MaxAttempts int
	// BackoffInitial is the base delay between attempts.
	BackoffInitial time.Duration
	// BackoffMax caps the delay.
	BackoffMax time.Duration
	// BackoffMultiplier multiplies the delay on each failure (default 2.0).
	BackoffMultiplier float64
	// RetryOnStatuses is a whitelist of status codes that should be retried.
	// Nil means retry on 502, 503, 504.
	RetryOnStatuses []int
	// RetryOnNetErr controls retry on transport errors (timeouts, resets).
	// Default true.
	RetryOnNetErr bool
}

// New constructs an *http.Client with the given defaults and optional
// retry behavior.
func New(cfg Config) *http.Client {
	applyDefaults(&cfg)

	transport := cfg.Transport
	if transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   cfg.DialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
			IdleConnTimeout:     cfg.IdleConnTimeout,
			MaxIdleConns:        cfg.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
			DisableKeepAlives:   cfg.DisableKeepAlives,
		}
	}

	if cfg.Retry.MaxAttempts > 1 {
		transport = &retryTransport{next: transport, cfg: cfg.Retry}
	}

	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}
}

func applyDefaults(c *Config) {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = 5 * time.Second
	}
	if c.TLSHandshakeTimeout == 0 {
		c.TLSHandshakeTimeout = 5 * time.Second
	}
	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = 90 * time.Second
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 100
	}
	if c.MaxIdleConnsPerHost == 0 {
		c.MaxIdleConnsPerHost = 10
	}
	if c.Retry.BackoffMultiplier == 0 {
		c.Retry.BackoffMultiplier = 2.0
	}
	if c.Retry.BackoffInitial == 0 {
		c.Retry.BackoffInitial = 100 * time.Millisecond
	}
	if c.Retry.BackoffMax == 0 {
		c.Retry.BackoffMax = 5 * time.Second
	}
	if !c.Retry.RetryOnNetErr {
		c.Retry.RetryOnNetErr = true
	}
	if c.Retry.RetryOnStatuses == nil {
		c.Retry.RetryOnStatuses = []int{
			http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout,
		}
	}
}

// retryTransport wraps an http.RoundTripper with retry logic.
type retryTransport struct {
	next http.RoundTripper
	cfg  RetryConfig
}

func (r *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Buffer body so we can replay it on retry.
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("httpx: read body for retry: %w", err)
		}
		_ = req.Body.Close()
		bodyBytes = b
	}

	var resp *http.Response
	var err error
	delay := r.cfg.BackoffInitial

	for attempt := 1; attempt <= r.cfg.MaxAttempts; attempt++ {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		resp, err = r.next.RoundTrip(req)

		if !r.shouldRetry(resp, err) || attempt == r.cfg.MaxAttempts {
			return resp, err
		}
		// Discard body between retries
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		if err := sleepCtx(req.Context(), delay); err != nil {
			return nil, err
		}
		delay = time.Duration(float64(delay) * r.cfg.BackoffMultiplier)
		if delay > r.cfg.BackoffMax {
			delay = r.cfg.BackoffMax
		}
	}
	return resp, err
}

func (r *retryTransport) shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		if !r.cfg.RetryOnNetErr {
			return false
		}
		// Retry on timeouts and net errors; don't retry on ctx cancel.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		return true
	}
	for _, s := range r.cfg.RetryOnStatuses {
		if resp.StatusCode == s {
			return true
		}
	}
	return false
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
