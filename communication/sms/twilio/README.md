# communication/sms/twilio

Twilio SMS adapter implementing `communication/provider.SMSProvider`.

```
go get github.com/mahdi-awadi/gopkg/communication/sms/twilio@latest
```

## Quickstart

```go
import (
    "context"
    twilsms "github.com/mahdi-awadi/gopkg/communication/sms/twilio"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := twilsms.New(twilsms.Config{
    AccountSID: "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    AuthToken:  "auth_token",
    FromNumber: "+15005550006",
}, nil /* or a Logger */)

resp, err := p.Send(context.Background(), &provider.SendRequest{
    RecipientPhone: "+12025550199",
    Body:           "Hello from gopkg",
})
```

## Notes

- Static credentials from Config; if any are empty, `Enabled()` returns false. Per-tenant runtime credentials can be layered via a wrapper.
- `SupportedCountries` / `CostEstimate` are overrideable via struct fields for accurate pricing.
- Use `MaskPhone` for safe logging of phone numbers.

## License

[MIT](../../../LICENSE)
