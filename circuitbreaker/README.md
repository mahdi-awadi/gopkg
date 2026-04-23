# circuitbreaker

[![Go Reference](https://pkg.go.dev/badge/github.com/mahdi-awadi/gopkg/circuitbreaker.svg)](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/circuitbreaker)

Three-state circuit breaker (closed → open → half-open → closed) with exponential backoff on repeated failures.

## Install

```bash
go get github.com/mahdi-awadi/gopkg/circuitbreaker
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/circuitbreaker"

b := circuitbreaker.New("downstream-api", circuitbreaker.DefaultConfig(), nil)

err := b.Execute(ctx, func(ctx context.Context) error {
    return apiCall(ctx)
}, nil) // nil → DefaultClassify

if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
    // fast-fail branch
}
```

## How it works

| State | Behavior |
|---|---|
| **Closed** | Every call passes through. Failures increment a counter. When `FailureThreshold` is reached, the breaker opens. |
| **Open** | Every call is rejected with `ErrCircuitOpen`. After `OpenTimeout`, the breaker transitions to half-open. |
| **Half-open** | Up to `HalfOpenMaxRequests` probe calls pass through. A probe success closes the breaker; a probe failure reopens it and multiplies `OpenTimeout` by `BackoffMultiplier` (capped at `MaxOpenTimeout`). |

## Error classification

Classify errors into short bucket strings (`"timeout"`, `"rate_limited"`, …) to distinguish downstream-health signals from caller-side problems. `Config.NonRetryableErrors` lists buckets that do NOT count against the failure budget.

- `DefaultClassify` is a substring-based best-effort classifier suitable for HTTP clients.
- Pass your own `ClassifyFunc` for typed-error structures (`errors.Is` / sentinels).

## Config

| Field | Default | Purpose |
|---|---|---|
| `FailureThreshold` | 5 | Failures to trip closed → open |
| `SuccessThreshold` | 2 | Successes to close half-open → closed |
| `OpenTimeout` | 30 s | Base wait before first half-open probe |
| `MaxOpenTimeout` | 1 h | Backoff cap (0 = unlimited) |
| `BackoffMultiplier` | 2.0 | Growth per failed probe (1.0/0 = disabled) |
| `HalfOpenMaxRequests` | 1 | Concurrent probes in half-open |
| `RetryableErrors` | (9 buckets) | Buckets that count against failure budget |
| `NonRetryableErrors` | (4 buckets) | Buckets that don't count (caller-side issues) |

## Observability

- `Breaker.Stats()` returns a snapshot (state, counts, current timeout, last failure).
- `Breaker.SetOnStateChange(fn)` fires a callback (in a goroutine) on every transition.
- Pass a `Logger` at construction time for structured transition logs.

## License

MIT © Mahdi Awadi
