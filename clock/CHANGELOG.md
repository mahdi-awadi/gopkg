# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Clock` interface (Now / After / Since)
- `Real{}` — zero-value production clock (time.Now / time.After)
- `Mock` — controllable test clock
  - `NewMock(start)` constructor
  - `Now/Since/After` implement Clock
  - `Advance(d)` fires waiters whose deadlines are due
  - `Set(t)` jumps to absolute time and fires due waiters
- 8 tests
- 1 runnable example (Mock.Advance + After)
- Zero third-party deps
