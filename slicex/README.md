# slicex

Generic slice helpers that complement the stdlib `slices` package. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/slicex@latest
```

## API

| Function | Type signature | Purpose |
|---|---|---|
| `Map[T, R](in, fn)` | `(in []T, fn func(T) R) []R` | Transform each element |
| `Filter[T](in, keep)` | `(in []T, keep func(T) bool) []T` | Keep matching elements |
| `Reduce[T, R](in, initial, fn)` | left-fold | Fold to a single value |
| `Unique[T comparable](in)` | `[]T` | Dedupe preserving order |
| `Chunk[T](in, n)` | `[][]T` | Split into size-n chunks |
| `GroupBy[T, K](in, keyFn)` | `map[K][]T` | Bucket by key |
| `Partition[T](in, keep)` | `(yes, no []T)` | Split by predicate |
| `Any / All / Find[T](in, pred)` | bool / bool / (T, bool) | Common predicates |

## Examples

```go
names := slicex.Map(users, func(u User) string { return u.Name })
adults := slicex.Filter(users, func(u User) bool { return u.Age >= 18 })
sum := slicex.Reduce([]int{1,2,3}, 0, func(a, b int) int { return a + b })
batches := slicex.Chunk(ids, 100)   // pagination over external APIs
```

## License

[MIT](../LICENSE)
