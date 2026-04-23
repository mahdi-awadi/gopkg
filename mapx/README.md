# mapx

Generic map helpers that complement stdlib `maps`. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/mapx@latest
```

## API

```go
mapx.Keys[K, V](m)              []K               // unordered
mapx.KeysSorted[K Ordered, V](m) []K              // ascending
mapx.Values[K, V](m)            []V

mapx.Merge[K, V](a, b, c, ...)  map[K]V           // later overrides
mapx.Invert[K, V comparable](m) map[V]K

mapx.Filter[K, V](m, keep)      map[K]V
mapx.MapValues[K, V, R](m, fn)  map[K]R
mapx.Equal[K, V comparable](a, b) bool            // nil == empty
```

## License

[MIT](../LICENSE)
