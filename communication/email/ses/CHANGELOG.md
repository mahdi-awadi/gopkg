# Changelog

## [Unreleased]

## [0.1.1] - 2026-04-23

### Fixed
- Drop leftover development-only `replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider` directive from `go.mod`. The replace was inert for downstream consumers (Go ignores `replace` in dependency modules), but it was noisy and contradicted the module's "publishable as-is" contract.

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.EmailProvider` via Amazon SES
- `Config{AccessKeyID, SecretAccessKey, Region, FromEmail, FromName}`
- `New(ctx, cfg, logger)` constructor (ctx for AWS config load)
- `Send`, `SendWithAttachments` (falls back to Send), `GetStatus`,
  `ValidateConfig`, `Enabled`, `Code`, `SupportedChannels`
- `MaskEmail` utility
- 2 tests (MaskEmail, zero-value not-Enabled)
- Uses aws-sdk-go-v2
