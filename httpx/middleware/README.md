# httpx/middleware

Small composable net/http middlewares: Recover, RequestID, Logger,
Timeout, CORS. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/httpx/middleware@latest
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/httpx/middleware"

stack := middleware.Chain(
    middleware.Recover(panicLogger),
    middleware.RequestID(""),
    middleware.Logger(func(e middleware.LogEntry) {
        log.Printf("%s %s %d (%v)", e.Method, e.Path, e.Status, e.Duration)
    }),
    middleware.Timeout(5 * time.Second),
    middleware.CORS(middleware.CORSConfig{AllowOrigins: []string{"https://example.com"}}),
)

http.Handle("/", stack(myRouter))
```

Chain wraps outermost first — the order you list them is the order
they receive the request.

## Components

### Recover
Catches panics, invokes `PanicLogger` (optional), returns 500.

### RequestID
Reuses inbound `X-Request-Id` header or generates 16-hex-char ID.
Stores on ctx; retrieve via `RequestIDFromContext(ctx)`.

### Logger
Calls `LogFunc(LogEntry)` once per request with method/path/status/duration/bytes/requestID.

### Timeout
Wraps with `http.TimeoutHandler`; returns 503 if handler doesn't finish.

### CORS
Simple origin allow-list + preflight handling. For regex origins or
per-route rules, use a dedicated library.

## License

[MIT](../../LICENSE)
