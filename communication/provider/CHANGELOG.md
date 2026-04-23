# Changelog

## [Unreleased]

## [0.1.0] - 2026-04-23

### Added
- `Channel` type + `ChannelEmail/SMS/Push/WhatsApp/Telegram/Voice` constants
- `Status` type + `StatusUnknown/Queued/Sent/Delivered/Read/Failed`
- `SendRequest`, `SendResponse`, `DeliveryStatus` value types
- `Provider` interface (6 methods)
- Channel-specific extension interfaces: `EmailProvider`, `SMSProvider`, `PushProvider`, `WhatsAppProvider`
- `Attachment` value type
- `ProviderError` with `errors.Is`/`errors.As` support + `NewProviderError` constructor
- `Registry` with `Register`, `Get`, `ByChannel`, `Codes`, `Len`
- Zero mandatory third-party dependencies
