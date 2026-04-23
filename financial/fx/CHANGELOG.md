# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Rates` interface (`GetRate(ctx, from, to)`, `Convert(ctx, amount, from, to)`)
- `Quote` struct (rate + `FetchedAt` metadata)
- `Memory` — concurrent-safe hot-swappable store
  - `Set`, `SetPair` (bidirectional), `Load` (atomic bulk replace), `Pairs` (defensive snapshot)
- `Static` — immutable table for tests/fixtures (input map is defensive-copied)
- `ErrRateNotFound`, `ErrInvalidRate` sentinels
- Same-currency short-circuit returning 1.0 (no lookup)
- Validation rejects zero/negative/Inf/NaN at Set/SetPair/Load/NewStatic
- Context-cancellation honored in GetRate/Convert
- 14 tests (same-currency, missing, bidirectional, bulk load preserves on error, Inf/NaN rejection, defensive-copy isolation, concurrency, immutable-static)
- 1 runnable example
- Zero third-party dependencies
