// Package metrics provides a thin Prometheus metrics exporter: an HTTP
// endpoint that serves your registry, plus optional Go + process
// collectors.
//
// The exporter does NOT ship a baked-in service-metrics struct;
// consumers register their own Prometheus collectors on the returned
// Registry. This keeps the package tiny and dependency-minimal.
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config configures the exporter.
type Config struct {
	// Port is the listening port for the dedicated metrics server.
	// Zero means do not start a server (you can still call Handler()).
	Port int
	// Path is the URL path served by Handler / ListenAndServe.
	// Default: "/metrics".
	Path string
	// EnableGoMetrics registers the standard Go runtime collector.
	EnableGoMetrics bool
	// EnableProcMetrics registers the standard process collector.
	EnableProcMetrics bool
}

// DefaultConfig returns a sensible default (Port, "/metrics", both collectors on).
func DefaultConfig(port int) Config {
	return Config{
		Port:              port,
		Path:              "/metrics",
		EnableGoMetrics:   true,
		EnableProcMetrics: true,
	}
}

// Exporter wraps a Prometheus registry and optional HTTP server.
//
// Use Registry() to register custom collectors. The exporter itself
// has no built-in business metrics.
type Exporter struct {
	cfg      Config
	registry *prometheus.Registry
	server   *http.Server
}

// New constructs an Exporter with a fresh Registry. Default collectors
// (Go + process) are registered per cfg.
func New(cfg Config) *Exporter {
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}
	reg := prometheus.NewRegistry()
	if cfg.EnableGoMetrics {
		reg.MustRegister(collectors.NewGoCollector())
	}
	if cfg.EnableProcMetrics {
		reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	return &Exporter{cfg: cfg, registry: reg}
}

// Registry returns the underlying *prometheus.Registry. Register your
// Counters / Histograms / Gauges here.
func (e *Exporter) Registry() *prometheus.Registry { return e.registry }

// Handler returns an http.Handler that serves the metrics endpoint.
func (e *Exporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		EnableOpenMetrics:   true,
		MaxRequestsInFlight: 10,
	})
}

// RegisterToMux mounts the metrics handler on an existing *http.ServeMux
// at cfg.Path.
func (e *Exporter) RegisterToMux(mux *http.ServeMux) {
	mux.Handle(e.cfg.Path, e.Handler())
}

// ListenAndServe starts a dedicated metrics HTTP server on cfg.Port.
// Returns an error on listen failure; blocks until Shutdown is called.
func (e *Exporter) ListenAndServe() error {
	if e.cfg.Port == 0 {
		return fmt.Errorf("metrics: Port is 0 — use RegisterToMux or set Port")
	}
	mux := http.NewServeMux()
	mux.Handle(e.cfg.Path, e.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	e.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", e.cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return e.server.ListenAndServe()
}

// StartBackground runs ListenAndServe in a goroutine. Returns a Shutdown
// function that gracefully stops the server.
func (e *Exporter) StartBackground() (shutdown func(context.Context) error) {
	go func() {
		if err := e.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Caller may log; package does not log.
			_ = err
		}
	}()
	return e.Shutdown
}

// Shutdown gracefully stops the metrics server.
func (e *Exporter) Shutdown(ctx context.Context) error {
	if e.server != nil {
		return e.server.Shutdown(ctx)
	}
	return nil
}
