# communication/push/fcm

Firebase Cloud Messaging implementation of `communication/provider.PushProvider`.

```
go get github.com/mahdi-awadi/gopkg/communication/push/fcm@latest
```

## Quickstart

```go
import (
    "context"
    fcm "github.com/mahdi-awadi/gopkg/communication/push/fcm"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p, err := fcm.New(ctx, fcm.Config{
    ProjectID:       "my-firebase-project",
    CredentialsJSON: string(serviceAccountBytes),
}, nil)
if err != nil { /* handle */ }

resp, err := p.Send(ctx, &provider.SendRequest{
    RecipientDeviceToken: "fcm-token...",
    Subject:              "Your order is ready",
    Body:                 "Tap to view details",
})

// Multicast to many tokens:
resp2, err := p.SendMulticast(ctx, []string{"t1","t2","t3"}, req)

// Topic broadcast:
resp3, err := p.SendToTopic(ctx, "promotions", req)
```

## Notes

- Sets sensible defaults (Android priority=high, APNS badge=1, sound=default)
- `GetStatus` returns `StatusSent`; real tracking requires Firebase Analytics
- Use `MaskToken` for safe logging

## License

[MIT](../../../LICENSE)
