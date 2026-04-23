# stringcase

Convert between `camelCase`, `PascalCase`, `snake_case`, `SCREAMING_SNAKE_CASE`, and `kebab-case`. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/stringcase@latest
```

## API

```go
stringcase.Snake("OrderItem")        // "order_item"
stringcase.ScreamingSnake("orderItem") // "ORDER_ITEM"
stringcase.Kebab("orderItem")        // "order-item"
stringcase.Camel("order_item")       // "orderItem"
stringcase.Pascal("order_item")      // "OrderItem"
stringcase.Split("HTTPRequest")      // ["HTTP", "Request"]
```

Handles acronym boundaries (`HTTPRequest` → `HTTP`+`Request`), underscores, hyphens, spaces, and digits. ASCII only.

## License

[MIT](../LICENSE)
