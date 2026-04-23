# identity/jwt

Minimal generic HMAC-SHA256 JWT wrapper around `github.com/golang-jwt/jwt/v5`.

```
go get github.com/mahdi-awadi/gopkg/identity/jwt@latest
```

## Quickstart

```go
import (
    "time"
    j "github.com/mahdi-awadi/gopkg/identity/jwt"
)

type MyClaims struct {
    UserID string `json:"uid"`
    Email  string `json:"email"`
    *j.RegisteredClaims
}

s, _ := j.New("very-secret-hmac-key")

// Issue a token
reg := j.StandardTTL("my-svc", "user-123", 15*time.Minute)
tok, _ := j.Issue(s, &MyClaims{UserID: "user-123", Email: "a@b.com", RegisteredClaims: &reg})

// Parse & verify
parsed, err := j.Parse(s, tok, &MyClaims{RegisteredClaims: &j.RegisteredClaims{}})
if errors.Is(err, jwt5.ErrTokenExpired) { /* handle */ }
```

## License

[MIT](../../LICENSE)
