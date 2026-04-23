# ratelimit

Thread-safe in-memory token-bucket limiter. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/ratelimit@latest
```

## Quickstart

```go
// 100 ops/sec, burst of 20.
l := ratelimit.New(20, 100)

if l.Allow() {
    // proceed
}

// Or wait with ctx cancellation:
if err := l.Wait(ctx); err != nil { /* ctx cancelled */ }
```

## Convenience: "one every X"

```go
l := ratelimit.NewEvery(10, time.Second) // 10 burst, 1/sec refill
```

## License

[MIT](../LICENSE)
