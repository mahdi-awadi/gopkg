# Changelog

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
