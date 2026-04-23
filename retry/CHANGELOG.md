# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Policy{MaxAttempts, InitialDelay, MaxDelay, Multiplier, JitterFraction}`
- `DefaultPolicy()` (5/100ms/5s/2x/25% jitter)
- `Do(ctx, policy, fn)` — context-aware retry loop
- `Permanent(err)` wrapper + `ErrPermanent` sentinel (errors.Is) to stop retries early
- Jitter is uniform random over `[d*(1-frac), d*(1+frac)]`
- ctx cancellation aborts mid-backoff
- 7 tests (success, retry-to-success, max-attempts, permanent, ctx cancel,
  clamp MaxAttempts<1, Permanent(nil))
- 2 runnable examples (with Output verification)
- Zero third-party deps
