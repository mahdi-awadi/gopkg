# Changelog

## [0.1.0] - 2026-04-23

### Added
- `New(Config) *http.Client` with defaults (30s total, 5s dial, 5s TLS, 100/10 pool, 90s idle)
- Optional retry transport: Config.Retry{MaxAttempts, BackoffInitial, BackoffMax, BackoffMultiplier, RetryOnStatuses, RetryOnNetErr}
- Default retryable statuses: 502, 503, 504
- Does NOT retry on ctx.Canceled / ctx.DeadlineExceeded
- Replays request body across retries (buffers once on first call)
- Swap in custom Transport via Config.Transport — retry wraps it
- 5 tests (defaults, no-retry on 200, retry-on-503, max-attempts respected,
  body replayed across retries, custom transport honored)
- 1 runnable example
- Zero third-party deps
