# communication/provider

Cross-channel notification-delivery contract for Go. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/communication/provider@latest
```

## What this is

A small set of interfaces and value types that every delivery adapter
(SMS, email, push, WhatsApp, voice, Telegram) satisfies. Build or import
adapters once; swap them freely.

## Core

```go
type Provider interface {
    Code() string
    SupportedChannels() []Channel
    Send(ctx context.Context, req *SendRequest) (*SendResponse, error)
    GetStatus(ctx context.Context, messageID string) (*DeliveryStatus, error)
    ValidateConfig() error
    Enabled() bool
}
```

Channel-specific extensions:

- `EmailProvider` adds `SendWithAttachments`
- `SMSProvider` adds `SupportedCountries`, `CostEstimate`
- `PushProvider` adds `SendToTopic`, `SendMulticast`
- `WhatsAppProvider` adds `SendTemplate`, `SendMedia`

## Registry

```go
r := provider.NewRegistry()
_ = r.Register(twilioSMS)
_ = r.Register(sendgrid)

p := r.Get("twilio")
smsProviders := r.ByChannel(provider.ChannelSMS)
```

## Errors

`ProviderError` is structured: `ProviderCode`, `Code`, `Message`, `Retryable`, and wraps a `RawError` (supports `errors.Is`/`errors.As`).

## License

[MIT](../../LICENSE)
