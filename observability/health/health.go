// Package health provides a small service health-check framework.
// Stdlib-only: exposes an http.Handler and an extensible check registry.
package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status is the top-level lifecycle state.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Result is the snapshot returned by Checker.Handler.
type Result struct {
	Status    Status            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks"`
}

// Check is an individual health check function.
// Returns nil on success, or an error describing the failure.
type Check func() error

// Checker aggregates named checks and renders them as JSON.
//
// Safe for concurrent use.
type Checker struct {
	service string
	version string

	mu     sync.RWMutex
	checks map[string]Check
}

// NewChecker returns a Checker for a given service name and version.
func NewChecker(service, version string) *Checker {
	return &Checker{
		service: service,
		version: version,
		checks:  make(map[string]Check),
	}
}

// Add registers a named check. Replaces any existing check with the same name.
func (c *Checker) Add(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

// Remove removes a named check (no-op if it doesn't exist).
func (c *Checker) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.checks, name)
}

// Evaluate runs all checks and returns the result. Sets Status to
// StatusUnhealthy if any check fails.
func (c *Checker) Evaluate() Result {
	c.mu.RLock()
	defer c.mu.RUnlock()

	r := Result{
		Status:    StatusHealthy,
		Timestamp: time.Now().UTC(),
		Service:   c.service,
		Version:   c.version,
		Checks:    make(map[string]string, len(c.checks)),
	}

	for name, fn := range c.checks {
		if err := fn(); err != nil {
			r.Checks[name] = err.Error()
			r.Status = StatusUnhealthy
		} else {
			r.Checks[name] = "ok"
		}
	}
	return r
}

// Handler returns an http.Handler that responds with the JSON health snapshot.
// Response code is 200 when healthy, 503 otherwise.
func (c *Checker) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := c.Evaluate()
		code := http.StatusOK
		if result.Status != StatusHealthy {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(result)
	})
}
