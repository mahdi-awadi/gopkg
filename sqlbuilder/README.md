# sqlbuilder

Tiny fluent SQL builder for parameterized Postgres SELECTs.

Solves the common duplication between COUNT and LIST variants of
paginated list/search endpoints: build the filters once, derive both
queries via `Clone`.

```
go get github.com/mahdi-awadi/gopkg/sqlbuilder@latest
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/sqlbuilder"

qb := sqlbuilder.New("users").
    WhereNotEmpty("name", req.Name).           // adds only if non-empty
    WhereOp("created_at", ">=", req.Since).    // custom operator
    WhereBool("is_active", true, req.OnlyActive). // gated
    WhereMultiLike(pattern, "name", "email").  // OR across columns
    OrderBy("created_at DESC")

countQ, countArgs := qb.Clone().BuildCountQuery()
listQ, listArgs   := qb.BuildListQuery("id, name, email", req.Limit, req.Page)

db.QueryRow(ctx, countQ, countArgs...).Scan(&total)
db.Select(ctx, &rows, listQ, listArgs...)
```

## API

| Method | Produces |
|---|---|
| `Where(col, v)` | `col = $N` |
| `WhereOp(col, op, v)` | `col op $N` |
| `WhereNotEmpty(col, s)` | skip if `s == ""` |
| `WhereNotNil(col, v)` | skip if `v == nil` |
| `WhereBool(col, v, gate)` | skip if `gate == false` |
| `WhereLike(col, pattern)` | `col ILIKE $N` |
| `WhereMultiLike(pattern, cols...)` | `(c1 ILIKE $N OR c2 ILIKE $N ...)` |
| `OrderBy(clause)` | raw ORDER BY — NEVER pass user input |
| `Clone()` | deep copy for separate count/list |
| `BuildCountQuery()` | returns SQL + args |
| `BuildListQuery(cols, limit, page)` | LIMIT/OFFSET from 1-based page |

## License

[MIT](../LICENSE)
