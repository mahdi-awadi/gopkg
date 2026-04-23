# bus/nats

NATS JetStream adapter for [bus.Broker](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/bus).

```
go get github.com/mahdi-awadi/gopkg/bus/nats@latest
```

## Quickstart

```go
import (
    "context"
    "github.com/mahdi-awadi/gopkg/bus"
    busnats "github.com/mahdi-awadi/gopkg/bus/nats"
)

b, err := busnats.NewBroker(&bus.Config{
    URL:         "nats://localhost:4222",
    User:        "optional",
    Password:    "optional",
    ServiceName: "my-service",
}, bus.NoopLogger{})
```

## Server requirements

- NATS server ≥ 2.10 with JetStream enabled
- Streams and consumers must be provisioned externally (this adapter does not create them)

## License

[MIT](../../LICENSE)
