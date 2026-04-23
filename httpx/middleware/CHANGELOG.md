# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Middleware` type + `Chain(mws ...)` composer
- `Recover(logFn)` — panic safety → 500
- `RequestID(headerName)` — echo-or-generate, stored on ctx; `RequestIDFromContext(ctx)` accessor; `HeaderRequestID` constant
- `Logger(logFn)` — captures method/path/status/duration/bytes/requestID per request
- `Timeout(d)` — wraps `http.TimeoutHandler`
- `CORS(cfg)` — origin allow-list + preflight handling
- `LogEntry`, `CORSConfig` value types
- 8 tests (Recover catches panic, RequestID generate+reuse, Logger captures,
  Chain ordering, CORS preflight + allow-list, Timeout 503)
- 1 runnable example showing full stack
- Zero third-party deps
