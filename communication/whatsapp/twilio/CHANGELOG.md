# Changelog

## [0.1.1] - 2026-04-23

### Fixed
- Drop leftover development-only `replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider` directive from `go.mod`. The replace was inert for downstream consumers (Go ignores `replace` in dependency modules), but it was noisy and contradicted the module's "publishable as-is" contract.

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.WhatsAppProvider`
- `Config{AccountSID, AuthToken, From, ContentSid}`
- `New(cfg, logger)` constructor
- `Send` (plain text), `SendTemplate` (Content API), `SendMedia`, `GetStatus`
- Auto-prefixes `whatsapp:` when needed
- `MaskPhone` utility
- Compile-time check
- 4 tests
