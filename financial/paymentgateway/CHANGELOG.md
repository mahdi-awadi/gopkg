# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Gateway` interface — Code/Name/SupportedCurrencies/SupportedMethods/Charge/Status/Cancel/Refund
- Domain-neutral request/response types: `PaymentRequest`, `PaymentResponse`, `RefundRequest`, `RefundResponse`, `StatusReport`
- `Status` vocabulary: `pending`, `authorized`, `captured`, `failed`, `cancelled`, `refunded`
- `GatewayConfig` opaque `map[string]any` for runtime secrets
- Sentinel errors: `ErrDeclined`, `ErrInsufficientFunds`, `ErrInvalidAmount`, `ErrUnsupportedMethod`, `ErrUnsupportedCurrency`, `ErrGatewayTimeout`, `ErrInvalidRequest`, `ErrNotFound`
- `BuildCallbackURLWithReference(baseURL, ref)` with query-string preservation
- `ExtractReferenceFromPath(path)` companion parser
- Amount convention: minor units (stripe-style), no floating-point
- 4 tests (build, extract, round-trip, sentinel uniqueness) + compile-time Gateway satisfaction check
- 1 runnable example
- Zero third-party dependencies
