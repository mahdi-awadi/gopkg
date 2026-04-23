# communication/email/ses

Amazon SES implementation of `communication/provider.EmailProvider`.

```
go get github.com/mahdi-awadi/gopkg/communication/email/ses@latest
```

## Quickstart

```go
import (
    "context"
    ses "github.com/mahdi-awadi/gopkg/communication/email/ses"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p, err := ses.New(ctx, ses.Config{
    AccessKeyID:     "AKIA...",
    SecretAccessKey: "...",
    Region:          "us-east-1",
    FromEmail:       "no-reply@example.com",
    FromName:        "Example",
}, nil)

resp, err := p.Send(ctx, &provider.SendRequest{
    RecipientEmail: "bob@example.com",
    Subject:        "Hi",
    Body:           "Plain text",
    HTMLBody:       "<p>HTML</p>",
})
```

## Notes

- Uses aws-sdk-go-v2.
- `SendWithAttachments` currently falls back to `Send` (no MIME encoding yet).
- `GetStatus` returns `StatusSent` — for real delivery tracking, consume SNS bounces/complaints externally.

## License

[MIT](../../../LICENSE)
