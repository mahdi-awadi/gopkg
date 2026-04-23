// Package jwt is a minimal generic wrapper around github.com/golang-jwt/jwt/v5
// for HMAC-SHA256 token sign/verify.
//
// Consumers define their own Claims struct (typically embedding
// jwt.RegisteredClaims for standard fields). Issue and Parse are
// generic over the Claims type, so every token is strongly-typed at
// the call site with no runtime casting.
package jwt

import (
	"fmt"
	"time"

	jwt5 "github.com/golang-jwt/jwt/v5"
)

// RegisteredClaims is re-exported for convenience so callers don't need
// to import the underlying library separately.
type RegisteredClaims = jwt5.RegisteredClaims

// NumericDate is re-exported for convenience.
type NumericDate = jwt5.NumericDate

// NewNumericDate is re-exported.
var NewNumericDate = jwt5.NewNumericDate

// Signer holds the HMAC secret. Zero value is invalid; use New.
type Signer struct {
	secret []byte
}

// New returns a Signer using the given HMAC secret.
// Returns an error if secret is empty.
func New(secret string) (*Signer, error) {
	if secret == "" {
		return nil, fmt.Errorf("jwt: empty secret")
	}
	return &Signer{secret: []byte(secret)}, nil
}

// Issue signs claims with HS256 and returns the compact token string.
// T must be a pointer type implementing jwt.Claims (typically a struct
// that embeds *RegisteredClaims).
func Issue[T jwt5.Claims](s *Signer, claims T) (string, error) {
	token := jwt5.NewWithClaims(jwt5.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// Parse parses and verifies a signed token into claims of type T.
// T must be a pointer to a struct implementing jwt.Claims.
//
// Returns the parsed claims on success. On failure, returns the
// underlying jwt library error (check via errors.Is against
// jwt.ErrTokenExpired / ErrSignatureInvalid / ErrTokenMalformed).
func Parse[T jwt5.Claims](s *Signer, tokenString string, empty T) (T, error) {
	token, err := jwt5.ParseWithClaims(tokenString, empty, func(t *jwt5.Token) (any, error) {
		if _, ok := t.Method.(*jwt5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	claims, ok := token.Claims.(T)
	if !ok || !token.Valid {
		var zero T
		return zero, fmt.Errorf("jwt: invalid token")
	}
	return claims, nil
}

// StandardTTL is a convenience wrapper constructing RegisteredClaims
// with reasonable defaults (IssuedAt=now, NotBefore=now, ExpiresAt=now+ttl).
func StandardTTL(issuer, subject string, ttl time.Duration) RegisteredClaims {
	now := time.Now()
	return RegisteredClaims{
		Issuer:    issuer,
		Subject:   subject,
		IssuedAt:  NewNumericDate(now),
		NotBefore: NewNumericDate(now),
		ExpiresAt: NewNumericDate(now.Add(ttl)),
	}
}
