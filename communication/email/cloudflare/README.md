# communication/email/cloudflare

Cloudflare Worker implementation of `communication/provider.EmailProvider`.

It sends email by POSTing a JSON body to a Cloudflare Worker HTTP endpoint. The
Worker holds the real mail credentials; this package only needs the Worker URL
and a shared secret. The secret is sent as `Authorization: Bearer <token>`.

```
go get github.com/mahdi-awadi/gopkg/communication/email/cloudflare@latest
```

## Quickstart

```go
import (
    "context"
    cf "github.com/mahdi-awadi/gopkg/communication/email/cloudflare"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := cf.New(cf.Config{
    WorkerURL: "https://mail.example.workers.dev/send",
    AuthToken: "worker-secret", // read from env/secret store, never hardcode
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

Attachments are sent base64-encoded in the JSON payload:

```go
resp, err := p.SendWithAttachments(ctx, req, []provider.Attachment{{
    Filename: "invoice.pdf", ContentType: "application/pdf", Content: pdfBytes,
}})
```

## Worker contract

The Worker receives:

```json
{
  "from": "no-reply@example.com",
  "from_name": "Example",
  "to": "bob@example.com",
  "subject": "Hi",
  "text": "Plain text body",
  "html": "<p>HTML body</p>",
  "attachments": [
    {"filename": "invoice.pdf", "content_type": "application/pdf", "content_base64": "..."}
  ]
}
```

On success it should return HTTP 2xx with a JSON body carrying a message id
(`{"id": "..."}` or `{"message_id": "..."}`). Any non-2xx response is reported
as a failed `SendResponse` with the status code and body in `Error`.

## License

[MIT](../../../LICENSE)
