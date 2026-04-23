# retry

Context-aware exponential backoff with jitter. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/retry@latest
```

## Quickstart

```go
import (
    "context"
    "github.com/mahdi-awadi/gopkg/retry"
)

err := retry.Do(ctx, retry.DefaultPolicy(), func() error {
    return callExternalAPI()
})
```

### Aborting on non-retriable errors

Wrap with `retry.Permanent(err)` to stop immediately:

```go
err := retry.Do(ctx, policy, func() error {
    resp, err := apiCall()
    if err != nil && resp.StatusCode == 403 {
        return retry.Permanent(err)   // do not retry
    }
    return err
})
if errors.Is(err, retry.ErrPermanent) { /* surfaced permanent */ }
```

### Tuning

```go
retry.Policy{
    MaxAttempts:    5,
    InitialDelay:   100 * time.Millisecond,
    MaxDelay:       5 * time.Second,
    Multiplier:     2.0,
    JitterFraction: 0.25,   // ±25% jitter
}
```

`ctx` cancellation aborts the retry loop immediately.

## License

[MIT](../LICENSE)
