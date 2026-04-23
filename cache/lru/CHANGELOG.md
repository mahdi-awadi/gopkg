# Changelog

## [0.1.0] - 2026-04-23

### Added
- Generic `Cache[K comparable, V any]` with Set/Get/Delete/Clear/Len
- `New(capacity, ttl)` — ttl 0 = no expiry
- Thread-safe via sync.Mutex
- Promotes on Get; evicts LRU on overflow
- TTL checked lazily on Get
- 7 tests (Set/Get, LRU eviction, TTL expiry, replace, Delete, Clear, min capacity)
- 1 runnable example (Output-verified)
- Zero third-party deps
