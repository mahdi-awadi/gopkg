# Changelog

## [0.1.0] - 2026-04-23

### Added
- `Email`, `PhoneE164`, `URL`, `UUID`, `Password`, `NormalizeEmail` functions
- `PasswordPolicy{MinLength, RequireUppercase/Lowercase/Digit/Symbol}`
- `DefaultPasswordPolicy()` helper
- 9 typed sentinels matchable via `errors.Is`
- 7 tests (email, phone, URL, UUID, password-each-branch, normalize)
- 4 runnable examples (Output-verified where deterministic)
- Zero third-party deps
