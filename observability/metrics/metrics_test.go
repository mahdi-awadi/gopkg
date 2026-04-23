package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNew_DefaultsApplied(t *testing.T) {
	e := New(Config{})
	if e.cfg.Path != "/metrics" {
		t.Fatalf("expected default path /metrics, got %q", e.cfg.Path)
	}
	if e.registry == nil {
		t.Fatal("registry is nil")
	}
}

func TestHandler_ExposesRegisteredMetric(t *testing.T) {
	e := New(Config{})
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gopkg_test_counter",
		Help: "test counter",
	})
	e.Registry().MustRegister(counter)
	counter.Add(42)

	rec := httptest.NewRecorder()
	e.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "gopkg_test_counter 42") {
		t.Fatalf("expected counter value in body, got:\n%s", body)
	}
}

func TestRegisterToMux(t *testing.T) {
	e := New(Config{Path: "/custom/metrics"})
	counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "mux_test_counter", Help: "x"})
	e.Registry().MustRegister(counter)

	mux := http.NewServeMux()
	e.RegisterToMux(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/custom/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListenAndServe_ReturnsErrorWhenPortZero(t *testing.T) {
	e := New(Config{Path: "/metrics"})
	if err := e.ListenAndServe(); err == nil {
		t.Fatal("expected error when Port=0")
	}
}

func TestGoAndProcCollectorsRegistered(t *testing.T) {
	e := New(Config{EnableGoMetrics: true, EnableProcMetrics: true})
	rec := httptest.NewRecorder()
	e.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, _ := io.ReadAll(rec.Body)
	s := string(body)
	if !strings.Contains(s, "go_goroutines") {
		t.Fatal("expected go_goroutines metric")
	}
	if !strings.Contains(s, "process_cpu_seconds_total") {
		t.Fatal("expected process_cpu_seconds_total metric")
	}
}
