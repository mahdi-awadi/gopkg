# Changelog

## [Unreleased]

## [0.1.0] - 2026-08-19

### Added
- `Provider` implementing `communication/provider.EmailProvider` over a
  Cloudflare Worker HTTP endpoint
- `Config{WorkerURL, AuthToken, FromEmail, FromName}`, `New(cfg, logger) *Provider`
- `Send`, `SendWithAttachments`, `GetStatus`, `ValidateConfig`, `Enabled`, `Code`,
  `SupportedChannels`
- JSON payload with base64-encoded attachments; `Authorization: Bearer <token>` auth
- Non-2xx responses surfaced as failed `SendResponse` with status code + body
- `MaskEmail` utility
- Compile-time check: `var _ provider.EmailProvider = (*Provider)(nil)`
- Table-driven tests (httptest worker stub) covering success, non-2xx, bad config,
  and attachments

### Dependencies
- gopkg/communication/provider v0.1.0
