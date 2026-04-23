# Changelog

## [0.1.0] - 2026-04-23

### Added
- `UUIDv7()` — 36-char hyphenated RFC 9562 v7 UUID
- `UUIDv7Raw()` — 16-byte binary form
- Time-ordered: top 48 bits = ms Unix timestamp
- Version nibble = 7, variant bits = 10 (per RFC 9562)
- Concurrent-safe via internal mutex
- 4 tests (regex format, 10k uniqueness, time-ordered lexical sort,
  raw bytes version/variant)
- 1 runnable example
- Zero third-party deps
