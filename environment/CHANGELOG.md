# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Environment` type + Development/Testing/Staging/Production constants
- `GetEnvironment()` reads `ENVIRONMENT` env var once (sync.Once cached)
- `IsDevelopment() / IsTesting() / IsStaging() / IsProduction()` predicates
- Defaults to Production for safety when unset/unrecognized
- 3 tests
- Zero third-party deps
