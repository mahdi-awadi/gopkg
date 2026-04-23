# observability/metrics

Thin Prometheus metrics exporter for Go services.

```
go get github.com/mahdi-awadi/gopkg/observability/metrics@latest
```

## Quickstart

```go
import (
    "context"
    "github.com/mahdi-awadi/gopkg/observability/metrics"
    "github.com/prometheus/client_golang/prometheus"
)

e := metrics.New(metrics.DefaultConfig(9090))

// Register your own metrics on the embedded registry:
reqCounter := prometheus.NewCounter(prometheus.CounterOpts{
    Name: "myservice_http_requests_total",
    Help: "HTTP request count",
})
e.Registry().MustRegister(reqCounter)

// Run the dedicated metrics server in the background:
shutdown := e.StartBackground()
defer shutdown(context.Background())

// Or mount on an existing mux:
// e.RegisterToMux(http.DefaultServeMux)
```

## Philosophy

This package provides the **transport** — a Prometheus HTTP endpoint, optional Go and process collectors, graceful shutdown — nothing else.

Domain metrics (e.g. `bookings_total`, `payments_total`) belong in the consuming service. Register them on `e.Registry()`.

## License

[MIT](../../LICENSE)
