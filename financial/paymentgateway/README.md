# financial/paymentgateway

[![Go Reference](https://pkg.go.dev/badge/github.com/mahdi-awadi/gopkg/financial/paymentgateway.svg)](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/financial/paymentgateway)

Canonical payment-gateway contract. Adapters (Stripe, Adyen, local rails, mocks) implement `Gateway` and ship in their own modules.

## Install

```bash
go get github.com/mahdi-awadi/gopkg/financial/paymentgateway
```

## The contract

```go
type Gateway interface {
    Code() string
    Name() string
    SupportedCurrencies() []string
    SupportedMethods() []string

    Charge(ctx, req PaymentRequest) (PaymentResponse, error)
    Status(ctx, gatewayPaymentID string) (StatusReport, error)
    Cancel(ctx, gatewayPaymentID string) error
    Refund(ctx, req RefundRequest) (RefundResponse, error)
}
```

## Amount representation

`Amount` is always in the currency's **minor unit** (cents, fils, pence, …). For zero-decimal currencies (JPY, IQD) that equals the major unit. Stripe-style, and removes every float-rounding footgun at the contract layer.

## Status vocabulary

`Status` is a small closed set: `pending`, `authorized`, `captured`, `failed`, `cancelled`, `refunded`. Gateway-specific status codes map onto one of these.

## Sentinel errors

Gateways wrap these with `%w` so callers can match with `errors.Is`:

- `ErrDeclined`
- `ErrInsufficientFunds`
- `ErrInvalidAmount`
- `ErrUnsupportedMethod`
- `ErrUnsupportedCurrency`
- `ErrGatewayTimeout`
- `ErrInvalidRequest`
- `ErrNotFound`

## Callback-URL helpers

Some gateways don't round-trip your identifiers through their callback payloads. `BuildCallbackURLWithReference` stuffs your reference into the URL path; `ExtractReferenceFromPath` pulls it back out on return.

```go
url := paymentgateway.BuildCallbackURLWithReference("https://x/cb?t=1", "order-42")
// https://x/cb/order-42?t=1

ref, _ := paymentgateway.ExtractReferenceFromPath(url)
// "order-42"
```

## License

MIT © Mahdi Awadi
