# validate

Small stateless input validators. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/validate@latest
```

## API

```go
validate.Email(s) error            // ErrInvalidEmail
validate.PhoneE164(s) error        // ErrInvalidPhone (strict E.164)
validate.URL(s) error              // ErrInvalidURL (absolute with host)
validate.UUID(s) error             // ErrInvalidUUID (RFC 9562 v1–v8)
validate.Password(s, policy) error // ErrPassword{TooShort,NoUpper,NoLower,NoDigit,NoSymbol}
validate.NormalizeEmail(s) string  // trim + lowercase
```

All error returns match via `errors.Is` against the package sentinels.

## Password policy

```go
p := validate.DefaultPasswordPolicy() // 8 chars + upper + lower + digit + symbol
p.MinLength = 12                      // override
err := validate.Password("MyStr0ng!pw", p)
```

## License

[MIT](../LICENSE)
