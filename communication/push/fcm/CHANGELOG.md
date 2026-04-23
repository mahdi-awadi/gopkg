# Changelog

## [Unreleased]

## [0.1.0] - 2026-04-23

### Added
- `Provider` implementing `communication/provider.PushProvider`
- `Config{ProjectID, CredentialsJSON}`, `New(ctx, cfg, logger) (*Provider, error)`
- `Send`, `SendToTopic`, `SendMulticast`, `GetStatus`, `ValidateConfig`,
  `Enabled`, `Code`, `SupportedChannels`
- Default Android (priority=high, sound=default) and APNS (sound=default, badge=1) configs
- `MaskToken` utility
- 3 tests (MaskToken, zero-value not-Enabled, ValidateConfig)
- Compile-time check: `var _ provider.PushProvider = (*Provider)(nil)`
