# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Pool` with fixed goroutine count
- `New(workers)` clamps workers < 1 to 1
- `Submit(fn)` — blocks if all workers busy
- `SubmitCtx(ctx, fn)` — ctx-aware
- `Wait()` — idempotent close + drain
- Panic recovery per task — one bad task does not kill a worker
- `ErrClosed` sentinel (matchable via errors.Is)
- 6 tests (all-tasks-run, Wait idempotent, Submit-after-Wait panics,
  SubmitCtx cancel, SubmitCtx closed, panic isolation)
- 1 Output-verified example
- Zero third-party deps
