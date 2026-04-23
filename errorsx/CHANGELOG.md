# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Kind` type + 9 kinds (NotFound/InvalidArgument/Conflict/Unauthenticated/
  PermissionDenied/FailedPrecondition/ResourceExhausted/Unavailable/DeadlineExceeded)
- `Error` type with `Kind()` + `Unwrap()` + `Is()` (matches sentinels by kind)
- `New(kind, msg)` / `Newf(kind, fmt, args...)`
- `Wrap(kind, err, msg)` (nil-safe)
- `KindOf(err)` — extracts Kind through unwrap chains
- `HTTPStatus(err)` — Kind → HTTP status code
- Sentinels: `ErrNotFound`, `ErrConflict`, etc. — use with errors.Is
- 7 tests (New/Newf/Wrap/Is/HTTPStatus/KindOf/nil handling)
- 2 runnable examples (Output-verified)
- Zero third-party deps
