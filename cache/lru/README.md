# cache/lru

Thread-safe generic LRU cache with optional TTL. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/cache/lru@latest
```

## Quickstart

```go
import (
    "time"
    "github.com/mahdi-awadi/gopkg/cache/lru"
)

// 1000 entries, 10-min TTL. ttl of 0 = no expiry (pure LRU).
c := lru.New[string, *User](1000, 10*time.Minute)

c.Set("user:42", user)
if u, ok := c.Get("user:42"); ok {
    // cache hit — automatically promoted to MRU
}
```

## API

- `New[K,V](capacity, ttl)`
- `Set(key, value)`, `Get(key) (V, bool)`, `Delete(key)`, `Clear()`, `Len()`

Zero third-party deps. Backed by stdlib `container/list`.

## License

[MIT](../../LICENSE)
