# Changelog

## [0.1.0] - 2026-04-23

### Added
- `NotifyContext(parent)` — cancels on SIGINT/SIGTERM
- `NotifyContextFor(parent, sigs...)` — custom signal set
- `Wait(ctx) os.Signal` / `WaitFor(ctx, sigs...)` — blocking variants
- `DefaultSignals()` — SIGINT + SIGTERM
- Signal-listener stopped via defer cancel (no leaked goroutines)
- 4 tests including real SIGINT/SIGTERM via syscall.Kill(getpid)
- 1 runnable example
- Zero third-party deps
