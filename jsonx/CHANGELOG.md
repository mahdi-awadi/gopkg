# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Decode(r, dst, opts)` — size-limited JSON body decoder
- `DecodeOptions{MaxBodySize, DisallowUnknownFields}`
- `DefaultMaxBodySize = 1 << 20` (1 MiB)
- `ErrTooLarge` typed error for size overflow
- Rejects trailing JSON documents in the body
- `Write(w, status, v)` — renders JSON with proper Content-Type, SetEscapeHTML(false)
- `Error(w, status, msg)` shorthand for {"error": msg}
- 7 tests covering all paths (ok, too-large, unknown-fields strict/loose,
  trailing content, status+CT, error shape, HTML not escaped)
- 1 runnable example showing a full handler
- Zero third-party deps
