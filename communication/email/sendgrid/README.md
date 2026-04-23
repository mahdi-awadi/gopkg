# communication/email/sendgrid

SendGrid implementation of `communication/provider.EmailProvider`.

```
go get github.com/mahdi-awadi/gopkg/communication/email/sendgrid@latest
```

## Quickstart

```go
import (
    "context"
    sg "github.com/mahdi-awadi/gopkg/communication/email/sendgrid"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := sg.New(sg.Config{
    APIKey:    "SG.xxx",
    FromEmail: "no-reply@example.com",
    FromName:  "Example",
}, nil /* or a Logger */)

resp, err := p.Send(context.Background(), &provider.SendRequest{
    RecipientEmail: "bob@example.com",
    Subject:        "Hi",
    Body:           "Plain text body",
    HTMLBody:       "<p>HTML body</p>",
})
```

### With attachments

```go
resp, err := p.SendWithAttachments(ctx, req, []provider.Attachment{{
    Filename: "invoice.pdf", ContentType: "application/pdf", Content: pdfBytes,
}})
```

## License

[MIT](../../../LICENSE)
