# Changelog

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
