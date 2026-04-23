# Changelog

## [Unreleased]

## [0.1.1] - 2026-04-23

### Fixed
- Drop leftover development-only `replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider` directive from `go.mod`. The replace was inert for downstream consumers (Go ignores `replace` in dependency modules), but it was noisy and contradicted the module's "publishable as-is" contract.

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.PushProvider`
- `New(logger) *Provider` — no credentials required
- `Send`, `SendMulticast`, `SendToTopic` (returns error — unsupported), `GetStatus`,
  `ValidateConfig`, `Enabled`, `Code`, `SupportedChannels`
- `PushMessage`, `PushTicket`, `PushResponse` types (for advanced callers)
- `IsExpoToken` helper
- `MaskToken` utility
- 6 tests (IsExpoToken, Enabled, Code/Channels, non-Expo token rejection,
  happy-path httptest, MaskToken)
- Zero mandatory third-party deps (stdlib-only)
