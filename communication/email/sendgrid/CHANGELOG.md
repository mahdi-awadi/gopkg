# Changelog

## [Unreleased]

## [0.1.1] - 2026-04-23

### Fixed
- Drop leftover development-only `replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider` directive from `go.mod`. The replace was inert for downstream consumers (Go ignores `replace` in dependency modules), but it was noisy and contradicted the module's "publishable as-is" contract.

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.EmailProvider`
- `Config{APIKey, FromEmail, FromName}`, `New(cfg, logger) *Provider`
- `Send`, `SendWithAttachments`, `GetStatus`, `ValidateConfig`, `Enabled`, `Code`, `SupportedChannels`
- `MaskEmail` utility
- Compile-time check: `var _ provider.EmailProvider = (*Provider)(nil)`
- 5 tests pass (constructor, Enabled variants, ValidateConfig, code/channels, MaskEmail)

### Dependencies
- gopkg/communication/provider v0.1.0
- github.com/sendgrid/sendgrid-go v3.14.0+incompatible
