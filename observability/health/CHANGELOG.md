# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Status` type + StatusHealthy/Unhealthy/Degraded
- `Result` JSON-serializable snapshot
- `Check` function type
- `Checker` with concurrent-safe `Add`/`Remove`/`Evaluate`/`Handler`
- `http.Handler` returns 200 on healthy, 503 otherwise
- 6 tests (no-checks baseline, failing/mixed checks, Remove, Handler 200/503)
- Zero third-party deps
