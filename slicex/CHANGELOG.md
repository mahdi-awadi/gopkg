# Changelog

## [0.1.0] - 2026-04-23

### Added
- Generic `Map[T, R]`, `Filter[T]`, `Reduce[T, R]`
- `Unique[T comparable]` — dedupe preserving order
- `Chunk[T]` — batch by size (panics on n <= 0)
- `GroupBy[T, K comparable]`
- `Partition[T]` — split by predicate
- `Any`, `All`, `Find[T]`
- 9 tests (one per function + edge cases)
- 5 Output-verified examples
- Zero third-party deps
