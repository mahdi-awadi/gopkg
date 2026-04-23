# signals

Context-based graceful shutdown on OS signals. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/signals@latest
```

## Quickstart

```go
ctx, cancel := signals.NotifyContext(context.Background())
defer cancel()

if err := server.Run(ctx); err != nil { /* ... */ }
// ctx is cancelled when SIGINT (Ctrl-C) or SIGTERM arrives.
```

### Custom signal set

```go
ctx, cancel := signals.NotifyContextFor(ctx, syscall.SIGUSR1, syscall.SIGHUP)
```

### Blocking variant

```go
sig := signals.Wait(ctx) // returns the received signal, or nil on ctx cancel
```

## License

[MIT](../LICENSE)
