# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Filter[T]` interface (Name/Criticality/Phase/Apply)
- `Criticality` enum (`Critical`, `Enrichment`)
- `Phase` int type (caller-defined values; no reserved semantics)
- `StepLog` per-step trace record (before/after count, duration, error, skipped)
- `Logger` interface (Debug/Warn/Error with `map[string]any` fields)
- `NoopLogger` zero-value logger
- `Pipeline[T]` with `New`, `Run`, `RunPhase`, `Filters`
- Defensive copy of filters at construction; `Filters()` returns a copy
- 10 tests covering pass-through, ordering, critical abort, enrichment skip, phase filter, meta passthrough, copy isolation
- 1 runnable example
- Zero third-party dependencies
