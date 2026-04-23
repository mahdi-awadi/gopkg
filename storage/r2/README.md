# storage/r2

Minimal Cloudflare R2 (S3-compatible object storage) client.

```
go get github.com/mahdi-awadi/gopkg/storage/r2@latest
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/storage/r2"

c, err := r2.New(&r2.Config{
    AccessKey:  "ACCESS_KEY",
    SecretKey:  "SECRET_KEY",
    Endpoint:   "https://<account-id>.r2.cloudflarestorage.com",
    Subdomain:  "files.example.com",   // optional, for public URL composition
    BucketName: "my-bucket",
})
if err != nil { /* handle */ }

url, err := c.Upload("avatars/123.png", pngBytes, "image/png")
signed, err := c.GetSignedURL("avatars/123.png", 5*time.Minute)
err = c.Delete("avatars/123.png")
```

## API

| Function | Purpose |
|---|---|
| `New(*Config) (*Client, error)` | Construct a client |
| `Client.Upload(key, content, contentType)` | PUT; returns public URL (empty if Subdomain not set) |
| `Client.GetSignedURL(key, expiry)` | Presigned GET URL |
| `Client.Delete(key)` | DELETE |

## License

[MIT](../../LICENSE)
