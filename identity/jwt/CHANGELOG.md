# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Signer` type + `New(secret)` constructor (errors on empty secret)
- Generic `Issue[T]` / `Parse[T]` functions — compile-time typed claims
- `RegisteredClaims` / `NumericDate` / `NewNumericDate` re-exports so
  consumers don't need a second import
- `StandardTTL(issuer, subject, ttl)` convenience helper
- 4 tests (empty secret, round-trip, wrong-secret rejection, expired-token)

### Dependencies
- github.com/golang-jwt/jwt/v5 v5.2.0
