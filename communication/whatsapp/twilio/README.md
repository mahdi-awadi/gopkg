# communication/whatsapp/twilio

WhatsApp-via-Twilio adapter implementing `communication/provider.WhatsAppProvider`.

```
go get github.com/mahdi-awadi/gopkg/communication/whatsapp/twilio@latest
```

## Quickstart

```go
import (
    "context"
    tw "github.com/mahdi-awadi/gopkg/communication/whatsapp/twilio"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := tw.New(tw.Config{
    AccountSID: "ACxxxx",
    AuthToken:  "token",
    From:       "+15005550006",  // adapter prefixes "whatsapp:" automatically
    ContentSid: "HXxxxx...",     // optional, for SendTemplate
}, nil)

// Plain text
resp, _ := p.Send(ctx, &provider.SendRequest{RecipientPhone: "+12025550199", Body: "Hi"})

// Template (uses Config.ContentSid, or override via Options["content_sid"])
resp2, _ := p.SendTemplate(ctx, &provider.SendRequest{RecipientPhone: "+1..."}, "irrelevant-name", []string{"Alice", "12345"})

// Media
resp3, _ := p.SendMedia(ctx, req, "https://example.com/image.jpg", "image/jpeg")
```

## License

[MIT](../../../LICENSE)
