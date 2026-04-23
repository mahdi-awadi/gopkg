# observability/health

Small stdlib-only service health-check framework with an `http.Handler`.

```
go get github.com/mahdi-awadi/gopkg/observability/health@latest
```

## Quickstart

```go
import (
    "database/sql"
    "net/http"
    "github.com/mahdi-awadi/gopkg/observability/health"
)

c := health.NewChecker("my-service", "v1.2.3")
c.Add("db", func() error { return db.Ping() })
c.Add("redis", func() error { return redisClient.Ping(ctx).Err() })

http.Handle("/health", c.Handler())
```

## API

- `NewChecker(service, version)` → `*Checker`
- `Add(name, Check)` / `Remove(name)`
- `Evaluate()` → `Result{Status, Timestamp, Service, Version, Checks}`
- `Handler()` → `http.Handler` (200 healthy / 503 unhealthy, JSON body)

## Zero third-party deps.

## License

[MIT](../../LICENSE)
