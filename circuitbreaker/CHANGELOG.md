# Changelog

## [0.1.0] - 2026-04-23

### Added
- Three-state Breaker (closed / open / half-open) with exponential backoff on repeated reopens
- `New(name, cfg, logger) *Breaker`
- `Allow()`, `Execute(ctx, fn, classify)`, `RecordSuccess`, `RecordFailure`, `Reset`
- `State()`, `Name()`, `Failures()`, `Stats()` for introspection
- `SetOnStateChange(fn)` transition callback (fires in a goroutine)
- `Logger` interface + `NoopLogger` zero value
- `ClassifyFunc` hook + `DefaultClassify` (substring-based fallback)
- `ErrCircuitOpen` sentinel
- `Config` with 8 tunable fields + `DefaultConfig()`
- `Config.IsRetryableError(bucket)` with NonRetryable > Retryable > default-true precedence
- 12 tests covering state transitions, backoff, caller-side vs provider-side errors, concurrent probes, callback firing
- 1 runnable example
- Zero third-party dependencies
