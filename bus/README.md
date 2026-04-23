# bus

Small, provider-agnostic pub/sub interface for Go.

```
go get github.com/mahdi-awadi/gopkg/bus@latest
```

## 30-second quickstart

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/mahdi-awadi/gopkg/bus"
    busnats "github.com/mahdi-awadi/gopkg/bus/nats"
)

func main() {
    ctx := context.Background()

    b, err := busnats.NewBroker(&bus.Config{
        URL:         "nats://localhost:4222",
        ServiceName: "orders",
    }, bus.NoopLogger{})
    if err != nil { log.Fatal(err) }
    defer b.Close()

    _ = b.PublishRaw(ctx, "orders.created", []byte(`{"id":"o_123"}`))

    sub, err := b.Subscribe(ctx, "ORDERS", "orders-worker", "orders.>", func(ctx context.Context, m bus.Message) error {
        fmt.Printf("%s: %s\n", m.Subject(), m.Data())
        return nil
    })
    if err != nil { log.Fatal(err) }
    defer sub.Stop()

    select {} // block on real use
}
```

## API

```go
type Broker interface {
    PublishRaw(ctx context.Context, subject string, data []byte, opts ...PublishOption) error
    Subscribe(ctx context.Context, topic, subscription, filter string, handler MessageHandler) (Subscription, error)
    Health() error
    Drain() error
    Close() error
}

type Logger interface {
    Info(msg string, fields map[string]any)
    Error(msg string, fields map[string]any)
}
```

- `bus.Message` — received message (Subject, Data)
- `bus.Subscription` — handle with `Stop()`
- `bus.PublishOption` — `WithOrderingKey(k)`, `WithHeaders(h)`
- `bus.NoopLogger{}` — zero-value discard logger
- `bus.WrapZap(*zap.Logger)` — convenience wrapper for `go.uber.org/zap`

## Adapters

| Adapter | Import | Backend |
|---|---|---|
| NATS JetStream | `github.com/mahdi-awadi/gopkg/bus/nats` | NATS ≥ 2.10 with JetStream |

## Versioning

`v0.x.y` pre-stable. Breaking changes may land on minor bumps.

## License

[MIT](../LICENSE)
