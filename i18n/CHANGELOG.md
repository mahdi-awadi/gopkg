# Changelog

## [0.1.0] - 2026-04-23

### Added
- `LocalizedString map[string]string` with `Get(lang) string`
- `sql.Scanner` + `driver.Valuer` for JSONB round-trip
- Fallback chain: exact lang → "en" → first non-empty → ""
- Nil/empty map → `{}` JSONB (never NULL)
- 7 tests covering Get fallbacks, Scan/Value round-trip, nil handling, string/byte
  variants, unsupported-type error, empty-lang argument
- Zero third-party deps
