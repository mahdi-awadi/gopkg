# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Registry[K comparable, V any]` — thread-safe generic key/value registry
- `New[K, V]() *Registry[K, V]`
- `Register(k, v) error` with `ErrAlreadyRegistered` on duplicate
- `Replace(k, v) (prev, replaced)` for intentional overwrite
- `Get(k) (v, error)` with `ErrNotFound` sentinel
- `Lookup(k) (v, ok)` allocation-free variant
- `Delete(k) bool`, `Keys() []K`, `Len() int`, `Range(fn func(k, v) bool)`
- `PendingQueue[K, V]` for init()-time deferred registration
- `PendingQueue.Add(fn)` queues, or runs immediately after Flush
- `PendingQueue.Flush(r) []error` runs queued fns and returns aligned error slice
- 14 tests covering duplicate-handling, concurrency, pending-queue edge cases
- 2 runnable examples (plain + pending-queue)
- Zero third-party dependencies
