# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.WhatsAppProvider` on Meta Cloud API v21
- `Config{PhoneNumberID, AccessToken, GraphAPIBase, Timeout}`
- `New(cfg, logger)` constructor
- `Send`, `SendTemplate`, `SendMedia`, `GetStatus`
- Template parameters as positional {{1}}, {{2}}, … via `parameters` slice
- Media types supported: image / video / audio / document
- `MessagesResponse`, `APIError` types for advanced callers
- `MaskPhone` utility
- 7 tests including happy-path httptest + API-error propagation
- Zero mandatory third-party deps (stdlib `net/http`)
