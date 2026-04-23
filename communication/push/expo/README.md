# communication/push/expo

Expo Push Notifications implementation of `communication/provider.PushProvider`.

```
go get github.com/mahdi-awadi/gopkg/communication/push/expo@latest
```

## Quickstart

```go
import (
    "context"
    expo "github.com/mahdi-awadi/gopkg/communication/push/expo"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := expo.New(nil)

resp, err := p.Send(context.Background(), &provider.SendRequest{
    RecipientDeviceToken: "ExponentPushToken[abc123]",
    Subject:              "Hello",
    Body:                 "Tap to view",
})
```

## Notes

- No credentials required (uses Expo's public push API at `https://exp.host`).
- `SendToTopic` returns an error — Expo public API doesn't support topics.
- `GetStatus` returns `StatusSent`; integrate Expo Receipts API for real delivery tracking.
- Helpers: `IsExpoToken`, `MaskToken`.

## License

[MIT](../../../LICENSE)
