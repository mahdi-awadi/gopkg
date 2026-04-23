# Changelog

## [0.1.0] - 2026-04-23

### Added
- `To[T](v) *T` — inline address-of
- `Deref[T](p) T` — *p or zero
- `Or[T](p, fallback) T` — *p or fallback
- `Equal[T comparable](a, b) bool` — nil-safe deep equality
- `IsNilOrZero[T comparable](p) bool`
- 5 tests + 3 Output-verified examples
- Zero third-party deps
