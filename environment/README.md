# environment

Tiny helper for the standard `ENVIRONMENT` env var.

```
go get github.com/mahdi-awadi/gopkg/environment@latest
```

## Usage

```go
import "github.com/mahdi-awadi/gopkg/environment"

if environment.IsDevelopment() {
    // ...
}
env := environment.GetEnvironment() // "development" | "testing" | "staging" | "production"
```

Reads `ENVIRONMENT` once on first call (sync.Once cached). Defaults to `production` if unset or unrecognized (safe default).

## Zero third-party deps.

## License

[MIT](../LICENSE)
