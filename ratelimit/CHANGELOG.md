# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Limiter` — token-bucket with Allow / Wait(ctx) / Tokens / Close
- `New(capacity, ratePerSecond)` constructor
- `NewEvery(capacity, interval)` convenience
- Starts with the bucket full
- Wait returns `ctx.Err()` on cancel, `ErrClosed` after Close
- 6 tests (burst + refill, capacity cap, ctx cancel, Close, NewEvery, bad args panic)
- 2 runnable examples (Output-verified)
- Zero third-party deps
