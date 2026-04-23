# i18n

`LocalizedString` — language-keyed string map with `sql.Scanner` + `driver.Valuer` for JSONB storage.

```
go get github.com/mahdi-awadi/gopkg/i18n@latest
```

## Usage

```go
import "github.com/mahdi-awadi/gopkg/i18n"

type Hotel struct {
    Name i18n.LocalizedString  // JSONB column in Postgres
}

h := Hotel{Name: i18n.LocalizedString{"en": "Hilton", "ar": "هيلتون"}}
h.Name.Get("ar")  // "هيلتون"
h.Name.Get("ku")  // "Hilton"  (fallback to en)
h.Name.Get("xx")  // "Hilton"  (fallback to en)
```

### Fallback order

1. Exact language match (non-empty value)
2. `"en"` key
3. First non-empty value in the map
4. `""` if nothing

### Database round-trip

Works with `pgx`/`database/sql` through the standard Scanner/Valuer interfaces.
Nil/empty maps serialize to `{}` (safe for NOT NULL JSONB columns).

## Zero third-party deps.

## License

[MIT](../LICENSE)
