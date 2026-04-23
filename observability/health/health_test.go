package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChecker_NoChecksIsHealthy(t *testing.T) {
	c := NewChecker("svc", "1.0")
	r := c.Evaluate()
	if r.Status != StatusHealthy {
		t.Fatalf("expected healthy, got %q", r.Status)
	}
	if len(r.Checks) != 0 {
		t.Fatalf("expected no checks, got %d", len(r.Checks))
	}
}

func TestChecker_FailingCheckMakesUnhealthy(t *testing.T) {
	c := NewChecker("svc", "1.0")
	c.Add("db", func() error { return errors.New("connection refused") })
	r := c.Evaluate()
	if r.Status != StatusUnhealthy {
		t.Fatalf("expected unhealthy, got %q", r.Status)
	}
	if r.Checks["db"] != "connection refused" {
		t.Fatalf("expected db=connection refused, got %q", r.Checks["db"])
	}
}

func TestChecker_MixedChecks(t *testing.T) {
	c := NewChecker("svc", "1.0")
	c.Add("db", func() error { return nil })
	c.Add("nats", func() error { return errors.New("down") })
	r := c.Evaluate()
	if r.Status != StatusUnhealthy {
		t.Fatalf("expected unhealthy, got %q", r.Status)
	}
	if r.Checks["db"] != "ok" {
		t.Fatalf("db should be ok, got %q", r.Checks["db"])
	}
}

func TestChecker_RemoveCheck(t *testing.T) {
	c := NewChecker("svc", "1.0")
	c.Add("x", func() error { return errors.New("x failed") })
	c.Remove("x")
	r := c.Evaluate()
	if r.Status != StatusHealthy {
		t.Fatalf("expected healthy after remove, got %q", r.Status)
	}
}

func TestHandler_200WhenHealthy(t *testing.T) {
	c := NewChecker("svc", "1.0")
	c.Add("ok", func() error { return nil })
	rec := httptest.NewRecorder()
	c.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result Result
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Service != "svc" || result.Version != "1.0" {
		t.Fatalf("service/version mismatch: %+v", result)
	}
}

func TestHandler_503WhenUnhealthy(t *testing.T) {
	c := NewChecker("svc", "1.0")
	c.Add("db", func() error { return errors.New("down") })
	rec := httptest.NewRecorder()
	c.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
