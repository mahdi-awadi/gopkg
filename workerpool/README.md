# workerpool

Fixed-size goroutine pool for bounded concurrency. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/workerpool@latest
```

## Quickstart

```go
p := workerpool.New(10)
for _, item := range items {
    item := item
    p.Submit(func() { process(item) })
}
p.Wait() // blocks until all submitted tasks finish
```

### With cancellation

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
if err := p.SubmitCtx(ctx, task); err != nil { /* ctx deadline / closed */ }
```

## Behavior

- Tasks that panic are recovered — one bad task does not kill a worker
- `Wait` is idempotent
- `Submit` after `Wait` panics; `SubmitCtx` returns `ErrClosed`

## License

[MIT](../LICENSE)
