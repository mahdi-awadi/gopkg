# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Config{Port, Path, EnableGoMetrics, EnableProcMetrics}`
- `New(cfg) *Exporter` — fresh Prometheus Registry + optional Go/proc collectors
- `Registry()` — register your own collectors
- `Handler()` → `http.Handler`
- `RegisterToMux(mux)` / `ListenAndServe()` / `StartBackground()` / `Shutdown(ctx)`
- 5 tests (defaults, registered metric exposed via Handler, RegisterToMux mount,
  Port=0 error, default collectors exposed)
- Transport-only: no domain metrics baked in

### Dependencies
- github.com/prometheus/client_golang v1.19.0
