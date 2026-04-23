# Changelog

## [0.1.1] - 2026-04-23

### Fixed
- Drop leftover development-only `replace github.com/mahdi-awadi/gopkg/communication/provider => ../../provider` directive from `go.mod`. The replace was inert for downstream consumers (Go ignores `replace` in dependency modules), but it was noisy and contradicted the module's "publishable as-is" contract.

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.Provider`
- `Config{BotToken, APIBaseURL, Timeout}` — static bot-token config
- `New(cfg, logger)`
- `Send` (text; HTML via SendRequest.HTMLBody → parse_mode=HTML), `GetStatus`,
  `ValidateConfig`, `Enabled`, `Code`, `SupportedChannels`
- `APIResponse` public type for advanced callers
- 5 tests (code/channels, ValidateConfig, no-chat-id failure,
  happy-path httptest, API-error propagation)
- Zero mandatory third-party deps (stdlib + gopkg/provider only)
