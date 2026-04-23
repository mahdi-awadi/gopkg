# Changelog

## [0.1.0] - 2026-04-23

### Added
- `DefaultCost = 12`
- `Hash(plain)` / `HashWithCost(plain, cost)`
- `Verify(hash, plain)` returns `ErrMismatch` on wrong password
- `NeedsRehash(hash, targetCost)` for rolling bcrypt-cost upgrades
- 4 tests + 1 Output-verified example

### Dependencies
- golang.org/x/crypto/bcrypt
