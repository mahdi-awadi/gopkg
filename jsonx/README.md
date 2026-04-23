# jsonx

Small JSON-over-HTTP helpers. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/jsonx@latest
```

## API

```go
// Decode request body, with size limit + optional strict-unknown-fields.
err := jsonx.Decode(r, &req, jsonx.DecodeOptions{
    MaxBodySize: 1 << 20,            // default 1 MiB
    DisallowUnknownFields: true,     // default false
})

// Write a typed response with status and proper Content-Type.
jsonx.Write(w, http.StatusCreated, resp)

// Shorthand JSON error response: {"error": "..."}.
jsonx.Error(w, http.StatusBadRequest, "invalid email")
```

## Features

- Size limit via `http.MaxBytesReader` (returns `ErrTooLarge` typed error)
- Optional strict decoding (`DisallowUnknownFields: true`)
- Rejects trailing JSON ("smuggled" second document)
- `Write` sets `Content-Type: application/json; charset=utf-8`
- `SetEscapeHTML(false)` so `<b>` renders literally (typical for APIs, not browser HTML)

## License

[MIT](../LICENSE)
