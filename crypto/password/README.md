# crypto/password

Thin bcrypt wrapper for password hashing + verification.

```
go get github.com/mahdi-awadi/gopkg/crypto/password@latest
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/crypto/password"

hash, err := password.Hash("hunter2")          // cost=12 default

err = password.Verify(hash, attempt)
if errors.Is(err, password.ErrMismatch) {
    // wrong password
}

// Upgrade-in-place: if the stored hash was computed at lower cost, rehash.
if ok, _ := password.NeedsRehash(hash, password.DefaultCost); ok {
    newHash, _ := password.Hash(attempt)
    // update stored hash...
}
```

## API

- `Hash(plain)` — cost=12
- `HashWithCost(plain, cost)`
- `Verify(hash, plain)` — `ErrMismatch` if wrong, other errors for malformed hash
- `NeedsRehash(hash, targetCost)` — helper for bumping bcrypt cost over time

## Dependencies

- `golang.org/x/crypto/bcrypt` (the only bcrypt game in town)

## License

[MIT](../../LICENSE)
