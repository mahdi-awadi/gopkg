# communication/voice/twilio

Twilio Voice adapter for flash-call notifications (OTP-style ring-once).

```
go get github.com/mahdi-awadi/gopkg/communication/voice/twilio@latest
```

## Quickstart

```go
import (
    "context"
    tv "github.com/mahdi-awadi/gopkg/communication/voice/twilio"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := tv.New(tv.Config{
    AccountSID: "ACxxxx",
    AuthToken:  "token",
    FromNumber: "+15005550006",
}, nil)

resp, err := p.Send(context.Background(), &provider.SendRequest{
    RecipientPhone: "+12025550199",
})
// Caller's phone rings once and is dropped — verification via the inbound-call event.
```

## License

[MIT](../../../LICENSE)
