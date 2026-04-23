// Package password is a thin bcrypt wrapper for hashing and verifying
// user passwords.
//
// Wraps golang.org/x/crypto/bcrypt with sensible defaults (cost=12).
// Not a new algorithm — just a friendlier API.
package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the bcrypt cost parameter used by Hash. 12 is a common
// 2025-era default balancing CPU cost and throughput.
const DefaultCost = 12

// ErrMismatch signals Verify found the password doesn't match the hash.
var ErrMismatch = errors.New("password: mismatch")

// Hash bcrypts the password with DefaultCost.
func Hash(plain string) (string, error) {
	return HashWithCost(plain, DefaultCost)
}

// HashWithCost hashes with a custom cost. bcrypt allows cost in [4, 31];
// the caller is responsible for picking a sane value.
func HashWithCost(plain string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify reports whether plain matches the stored hash. Returns
// ErrMismatch when the password does not match (matchable via errors.Is).
// Other errors (malformed hash) are returned as-is.
func Verify(hash, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrMismatch
	}
	return err
}

// NeedsRehash reports whether a stored hash should be re-computed at
// the current cost (typically because DefaultCost was increased since
// the hash was generated).
func NeedsRehash(hash string, targetCost int) (bool, error) {
	c, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false, err
	}
	return c < targetCost, nil
}
