# ptr

Tiny generic helpers for `*T` nullability. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/ptr@latest
```

## API

```go
ptr.To(v) *T            // &v, inline
ptr.Deref(p) T          // *p, or zero if nil
ptr.Or(p, fallback) T   // *p, or fallback if nil
ptr.Equal(a, b) bool    // *T comparable deep-equality, nil-safe
ptr.IsNilOrZero(p) bool // nil OR *p == zero value
```

## License

[MIT](../LICENSE)
