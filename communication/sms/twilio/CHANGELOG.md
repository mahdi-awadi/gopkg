# Changelog

## [Unreleased]

## [0.1.1] - 2026-04-23

### Fixed
- Drop leftover development-only `replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider` directive from `go.mod`. The replace was inert for downstream consumers (Go ignores `replace` in dependency modules), but it was noisy and contradicted the module's "publishable as-is" contract.

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.SMSProvider`
- `Config` struct (AccountSID, AuthToken, FromNumber)
- `New(cfg, logger)` constructor with nil-logger noop fallback
- `SupportedChannels`, `Code`, `Enabled`, `Send`, `GetStatus`, `SupportedCountries`, `CostEstimate`, `ValidateConfig`
- Overridable `Countries` and `CountryPricing` struct fields
- `DefaultCountries()`, `DefaultCountryPricing()` helpers
- `MaskPhone` utility
- Compile-time check: `var _ provider.SMSProvider = (*Provider)(nil)`
- 7 unit tests (constructor, Enabled, Code, SupportedChannels, CostEstimate known + fallback, MaskPhone)

### Dependencies
- `github.com/mahdi-awadi/gopkg/communication/provider` v0.1.0
- `github.com/twilio/twilio-go` v1.20.0
