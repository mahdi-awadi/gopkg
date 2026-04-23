// Package fx defines the foreign-exchange interface used across
// gopkg financial packages and ships two reference implementations:
// Memory (in-process table, hot-swappable) and Static (immutable,
// constructed once).
//
// The interface is the canonical dependency for business code;
// production deployments inject an implementation backed by a
// database, a vendor API, or a rates-cache service.
//
// The package is zero-dependency and safe for concurrent use.
package fx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// ErrRateNotFound is returned when the requested currency pair is
// not available in the underlying store.
var ErrRateNotFound = errors.New("fx: rate not found")

// ErrInvalidRate is returned on any non-finite / non-positive rate
// value encountered during construction or update.
var ErrInvalidRate = errors.New("fx: invalid rate (must be finite and > 0)")

// Rates is the canonical foreign-exchange interface.
//
// All methods honor ctx cancellation. Implementations MUST return
// 1.0 for a same-currency pair. GetRate returns ErrRateNotFound for
// unknown pairs; Convert returns ErrRateNotFound only when the
// underlying rate lookup fails.
type Rates interface {
	// GetRate returns the multiplicative rate that converts 1 unit of
	// `from` into `to` units of `to`. Example: GetRate("USD","IQD")=1500
	// means 1 USD buys 1500 IQD.
	GetRate(ctx context.Context, from, to string) (float64, error)

	// Convert is the convenience form: amount * rate(from→to).
	Convert(ctx context.Context, amount float64, from, to string) (float64, error)
}

// Quote captures a rate with metadata about when it was sourced.
// Useful for logging, invalidation, and price-freshness gates.
type Quote struct {
	From      string
	To        string
	Rate      float64
	FetchedAt time.Time
}

// Memory is an in-process, mutex-guarded Rates implementation that
// can be hot-swapped (Set) and bulk-loaded (Load). Safe for concurrent
// use. The zero value is empty (every lookup returns ErrRateNotFound
// except same-currency pairs).
type Memory struct {
	mu    sync.RWMutex
	rates map[string]float64 // key: "FROM>TO"
}

// NewMemory returns an empty Memory store.
func NewMemory() *Memory {
	return &Memory{rates: make(map[string]float64)}
}

// Set installs or overwrites a single directional rate.
// Returns ErrInvalidRate if rate is not finite and > 0.
func (m *Memory) Set(from, to string, rate float64) error {
	if !validRate(rate) {
		return fmt.Errorf("%w: %s→%s = %v", ErrInvalidRate, from, to, rate)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rates[key(from, to)] = rate
	return nil
}

// SetPair installs both directions of a pair at once:
// from→to = rate, to→from = 1/rate. Convenient for bi-directional pairs.
func (m *Memory) SetPair(from, to string, rate float64) error {
	if !validRate(rate) {
		return fmt.Errorf("%w: %s→%s = %v", ErrInvalidRate, from, to, rate)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rates[key(from, to)] = rate
	m.rates[key(to, from)] = 1.0 / rate
	return nil
}

// Load bulk-replaces the entire rate table atomically. Keys are
// directional pairs "FROM>TO"; values are the multiplicative rates.
// Returns ErrInvalidRate if any value is invalid; the existing table
// is preserved on error.
func (m *Memory) Load(rates map[string]float64) error {
	// Validate first before touching the mutex-protected state.
	for k, v := range rates {
		if !validRate(v) {
			return fmt.Errorf("%w: %s = %v", ErrInvalidRate, k, v)
		}
	}
	next := make(map[string]float64, len(rates))
	for k, v := range rates {
		next[k] = v
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rates = next
	return nil
}

// GetRate implements Rates.
func (m *Memory) GetRate(ctx context.Context, from, to string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if from == to {
		return 1.0, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rates[key(from, to)]
	if !ok {
		return 0, fmt.Errorf("%w: %s→%s", ErrRateNotFound, from, to)
	}
	return r, nil
}

// Convert implements Rates.
func (m *Memory) Convert(ctx context.Context, amount float64, from, to string) (float64, error) {
	rate, err := m.GetRate(ctx, from, to)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

// Pairs returns a snapshot of the current rate table.
// Useful for inspection and persistence.
func (m *Memory) Pairs() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]float64, len(m.rates))
	for k, v := range m.rates {
		out[k] = v
	}
	return out
}

// Static is an immutable Rates implementation — build once, pass
// around, never changes. Preferable to Memory for tests and fixtures
// where mutability is a liability.
type Static struct {
	rates map[string]float64
}

// NewStatic constructs an immutable store from a directional pair map.
// Returns ErrInvalidRate if any value is invalid.
func NewStatic(rates map[string]float64) (*Static, error) {
	for k, v := range rates {
		if !validRate(v) {
			return nil, fmt.Errorf("%w: %s = %v", ErrInvalidRate, k, v)
		}
	}
	cp := make(map[string]float64, len(rates))
	for k, v := range rates {
		cp[k] = v
	}
	return &Static{rates: cp}, nil
}

// GetRate implements Rates.
func (s *Static) GetRate(ctx context.Context, from, to string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if from == to {
		return 1.0, nil
	}
	r, ok := s.rates[key(from, to)]
	if !ok {
		return 0, fmt.Errorf("%w: %s→%s", ErrRateNotFound, from, to)
	}
	return r, nil
}

// Convert implements Rates.
func (s *Static) Convert(ctx context.Context, amount float64, from, to string) (float64, error) {
	rate, err := s.GetRate(ctx, from, to)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

func key(from, to string) string { return from + ">" + to }

func validRate(r float64) bool { return r > 0 && !math.IsInf(r, 0) && !math.IsNaN(r) }
