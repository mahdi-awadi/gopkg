# Changelog

## [0.1.1] - 2026-04-23

### Fixed
- Drop leftover development-only `replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider` directive from `go.mod`. The replace was inert for downstream consumers (Go ignores `replace` in dependency modules), but it was noisy and contradicted the module's "publishable as-is" contract.

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.Provider` for flash-call voice
- `Config{AccountSID, AuthToken, FromNumber, RejectURL}`
- `New(cfg, logger) *Provider`
- `Send` places a call using `DefaultRejectURL` (or override) for ring-once behavior
- `GetStatus` maps Twilio call status to `provider.Status`
- `MaskPhone` utility
- Compile-time check: `var _ provider.Provider = (*Provider)(nil)`
- 4 tests
