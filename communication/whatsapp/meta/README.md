# communication/whatsapp/meta

WhatsApp-via-Meta-Cloud-API adapter implementing `communication/provider.WhatsAppProvider`.

```
go get github.com/mahdi-awadi/gopkg/communication/whatsapp/meta@latest
```

## Quickstart

```go
import (
    "context"
    meta "github.com/mahdi-awadi/gopkg/communication/whatsapp/meta"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := meta.New(meta.Config{
    PhoneNumberID: "1234567890",
    AccessToken:   "EAAG...",
}, nil)

// Text (only allowed inside 24h service window)
resp, _ := p.Send(ctx, &provider.SendRequest{
    RecipientPhone: "+12025550199",
    Body:           "Hello",
})

// Template (use when outside service window)
resp2, _ := p.SendTemplate(ctx, &provider.SendRequest{
    RecipientPhone: "+12025550199",
    Language:       "en_US",
}, "order_confirmation", []string{"Alice", "ORD-1234", "$42.00"})

// Media
resp3, _ := p.SendMedia(ctx, &provider.SendRequest{
    RecipientPhone: "+12025550199",
    Body:           "Your receipt",
}, "https://example.com/receipt.pdf", "document")
```

## Notes

- Zero third-party deps (stdlib `net/http` only).
- `Config.GraphAPIBase` can be overridden for testing against a fake server.
- Default API version: `v21.0`.
- `GetStatus` returns `StatusSent`; real delivery/read tracking comes in via Webhook events (handle those externally).

## License

[MIT](../../../LICENSE)
