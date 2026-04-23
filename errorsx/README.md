# errorsx

Tiny error taxonomy with HTTP status mapping. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/errorsx@latest
```

## Kinds

Maps to HTTP (and easily gRPC) codes:

| Kind | HTTP |
|---|---|
| `KindNotFound` | 404 |
| `KindInvalidArgument` | 400 |
| `KindConflict` | 409 |
| `KindUnauthenticated` | 401 |
| `KindPermissionDenied` | 403 |
| `KindFailedPrecondition` | 412 |
| `KindResourceExhausted` | 429 |
| `KindUnavailable` | 503 |
| `KindDeadlineExceeded` | 504 |
| `KindUnknown` (default) | 500 |

## Usage

```go
import "github.com/mahdi-awadi/gopkg/errorsx"

func GetUser(id string) (*User, error) {
    row, err := db.QueryRow(...)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, errorsx.New(errorsx.KindNotFound, "user not found")
    }
    if err != nil {
        return nil, errorsx.Wrap(errorsx.KindUnavailable, err, "db query failed")
    }
    ...
}

// HTTP handler:
if err := svc.DoThing(...); err != nil {
    http.Error(w, err.Error(), errorsx.HTTPStatus(err))
}
```

## Matching

`errors.Is` against the package sentinels matches by Kind:

```go
if errors.Is(err, errorsx.ErrNotFound) { ... }
if errorsx.KindOf(err) == errorsx.KindConflict { ... }
```

## License

[MIT](../LICENSE)
