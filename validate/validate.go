// Package validate provides small stateless validators for common
// user-input shapes: email, phone (E.164), URL, UUID, and password rules.
//
// Returns typed errors so callers can do `errors.Is(err, validate.ErrInvalidEmail)`.
//
// Zero third-party deps.
package validate

import (
	"errors"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// Typed errors. Match via errors.Is.
var (
	ErrInvalidEmail    = errors.New("validate: invalid email")
	ErrInvalidPhone    = errors.New("validate: invalid phone (E.164 required)")
	ErrInvalidURL      = errors.New("validate: invalid URL")
	ErrInvalidUUID     = errors.New("validate: invalid UUID")
	ErrPasswordTooShort = errors.New("validate: password too short")
	ErrPasswordNoUpper  = errors.New("validate: password missing uppercase letter")
	ErrPasswordNoLower  = errors.New("validate: password missing lowercase letter")
	ErrPasswordNoDigit  = errors.New("validate: password missing digit")
	ErrPasswordNoSymbol = errors.New("validate: password missing symbol")
)

// Email validates an RFC 5321 addr-spec. Returns ErrInvalidEmail if not parseable.
func Email(s string) error {
	if _, err := mail.ParseAddress(s); err != nil {
		return ErrInvalidEmail
	}
	return nil
}

// phoneE164 is a strict E.164 pattern: leading '+', 1–15 digits,
// first digit 1–9 (no leading-zero country codes).
var phoneE164 = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// PhoneE164 validates the E.164 international format.
func PhoneE164(s string) error {
	if !phoneE164.MatchString(s) {
		return ErrInvalidPhone
	}
	return nil
}

// URL validates an absolute URL with scheme and host.
func URL(s string) error {
	u, err := url.Parse(s)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

// uuidRe matches any RFC 9562 UUID (v1-v8) with hyphens.
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// UUID validates the string form of any RFC 9562 UUID version 1-8.
func UUID(s string) error {
	if !uuidRe.MatchString(s) {
		return ErrInvalidUUID
	}
	return nil
}

// PasswordPolicy controls password validation.
type PasswordPolicy struct {
	MinLength         int
	RequireUppercase  bool
	RequireLowercase  bool
	RequireDigit      bool
	RequireSymbol     bool
}

// DefaultPasswordPolicy is 8 chars + upper + lower + digit + symbol.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSymbol:    true,
	}
}

// Password validates p against the policy. Returns the first failing
// sentinel (ErrPasswordTooShort / NoUpper / NoLower / NoDigit / NoSymbol).
func Password(p string, policy PasswordPolicy) error {
	if len(p) < policy.MinLength {
		return ErrPasswordTooShort
	}
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if policy.RequireUppercase && !hasUpper {
		return ErrPasswordNoUpper
	}
	if policy.RequireLowercase && !hasLower {
		return ErrPasswordNoLower
	}
	if policy.RequireDigit && !hasDigit {
		return ErrPasswordNoDigit
	}
	if policy.RequireSymbol && !hasSymbol {
		return ErrPasswordNoSymbol
	}
	return nil
}

// NormalizeEmail lowercases and trims. Useful before equality checks or storage.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
