# clock

Tiny time-source abstraction for testability. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/clock@latest
```

## Why

Code that calls `time.Now()` and `time.After(...)` is hard to test because
real-world seconds have to elapse. With `clock.Clock` as a dependency, the
same code uses a `*Mock` in tests — `Advance(d)` jumps time instantly.

## Production

```go
type Cache struct { clock clock.Clock }

func New() *Cache { return &Cache{clock: clock.Real{}} }
```

## Tests

```go
m := clock.NewMock(time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC))
cache := &Cache{clock: m}

cache.Set("k", "v", 1*time.Hour)
m.Advance(30 * time.Minute)
ok := cache.Get("k") // still present

m.Advance(31 * time.Minute)
ok = cache.Get("k") // expired
```

## API

```go
type Clock interface {
    Now() time.Time
    After(d time.Duration) <-chan time.Time
    Since(t time.Time) time.Duration
}

type Real struct{}         // backed by time.Now
type Mock struct{...}      // controllable in tests
    NewMock(start) *Mock
    (m).Now/Since/After    // implements Clock
    (m).Advance(d)         // fires due timers
    (m).Set(t)             // jump to absolute time
```

## License

[MIT](../LICENSE)
