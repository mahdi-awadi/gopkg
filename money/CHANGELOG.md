# Changelog

## [0.1.0] - 2026-04-23

### Added
- Immutable `Money` type (int64 minor units + Currency)
- `Currency` type backed by ISO 4217 codes
- Pre-registered common currencies with correct scale (USD/EUR/GBP=2, JPY/KRW/VND/IQD=0, KWD/BHD/OMR/JOD=3, etc.)
- `Register(code, scale)` for custom currencies (scale 0–8)
- `New(amount, currency)`, `FromMinor(minor, currency)` constructors
- Operations: `Add`, `Sub`, `MulInt`, `DivInt`, `Negate`, `Equal`, `Cmp`, `IsZero`, `Minor`, `Currency`
- `String()` — formatted with correct decimals per currency
- Overflow detection on Add/Sub/MulInt
- `ErrCurrencyMismatch`, `ErrDivideByZero`, `ErrOverflow`, `ErrUnknownCurrency`, `ErrInvalidFormat`
- 13 tests + 3 runnable examples (Output-verified)
- Zero third-party deps
