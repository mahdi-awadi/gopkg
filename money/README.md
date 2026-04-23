# money

Immutable `Money` value type with explicit currency and minor-unit
precision. No floats. Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/money@latest
```

## Why

Commerce code that uses `float64` for money will eventually lose a cent to
rounding. `money.Money` stores integer minor units (cents, fils, qirsh)
and consults a per-currency scale table to format and parse.

## Usage

```go
import "github.com/mahdi-awadi/gopkg/money"

price, _ := money.New(9.99, "USD")      // 999 cents
tax, _   := money.New(0.85, "USD")
total, _ := price.Add(tax)              // 10.84 USD
fmt.Println(total)                      // "10.84 USD"

// Minor-unit input for APIs that give you raw integers:
kwd, _ := money.FromMinor(12345, "KWD") // 12.345 KWD (3-decimal currency)
```

## Supported currencies

Common ISO 4217 codes ship pre-registered with the correct scale
(USD=2, KWD=3, IQD=0, JPY=0, etc.). Register custom codes:

```go
_ = money.Register("MYCOIN", 4) // 4 decimal places
```

## Operations

- `Add`, `Sub` — same-currency; returns `ErrCurrencyMismatch` otherwise
- `MulInt(factor)`, `DivInt(divisor)` — integer multipliers (no precision loss)
- `Cmp(other) int` — -1/0/+1
- `Equal`, `IsZero`, `Negate`, `Minor`, `Currency`
- `String()` — formatted with correct decimals for the currency

## License

[MIT](../LICENSE)
