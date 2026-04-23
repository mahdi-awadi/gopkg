# httpx/client

A better default `*http.Client` — sane timeouts, pooled connections, optional retry on transient errors. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/httpx/client@latest
```

## Quickstart

```go
import (
    "time"
    "github.com/mahdi-awadi/gopkg/httpx/client"
)

c := client.New(client.Config{
    Timeout: 10 * time.Second,
    Retry: client.RetryConfig{
        MaxAttempts:    5,
        BackoffInitial: 200 * time.Millisecond,
        BackoffMax:     3 * time.Second,
    },
})

resp, err := c.Get("https://api.example.com/things")
```

## What it does

- Applies sensible defaults: 30s total timeout, 5s dial, 5s TLS handshake, 100/10 pooled idle conns, 90s idle timeout
- Optional retry transport wraps your normal transport — transparent to downstream code
- Replays request body on retry (buffers once, reuses for each attempt)
- Does NOT retry on ctx cancellation / deadline — treats those as final
- Default retry status codes: 502, 503, 504 (override via `RetryOnStatuses`)

## Advanced

Swap in your own transport (for OTel instrumentation, circuit breakers, etc.) via `Config.Transport` — the retry wrapper still layers on top if `Retry.MaxAttempts > 1`.

## License

[MIT](../../LICENSE)
