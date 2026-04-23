package password

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerify_RoundTrip(t *testing.T) {
	// Use minimum cost to keep the test fast.
	hash, err := HashWithCost("hunter2", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("hash should start with $2: %q", hash)
	}
	if err := Verify(hash, "hunter2"); err != nil {
		t.Fatalf("correct password should Verify, got %v", err)
	}
}

func TestVerify_WrongPasswordReturnsErrMismatch(t *testing.T) {
	hash, _ := HashWithCost("hunter2", bcrypt.MinCost)
	err := Verify(hash, "wrong")
	if !errors.Is(err, ErrMismatch) {
		t.Fatalf("expected ErrMismatch, got %v", err)
	}
}

func TestVerify_MalformedHashReturnsUnderlyingError(t *testing.T) {
	err := Verify("not-a-bcrypt-hash", "hunter2")
	if err == nil {
		t.Fatal("expected non-nil error on malformed hash")
	}
	if errors.Is(err, ErrMismatch) {
		t.Fatal("malformed hash should NOT be ErrMismatch")
	}
}

func TestNeedsRehash(t *testing.T) {
	hash, _ := HashWithCost("pw", bcrypt.MinCost)

	needs, err := NeedsRehash(hash, DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if !needs {
		t.Fatal("lower-cost hash should need rehash at DefaultCost")
	}

	needs, err = NeedsRehash(hash, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if needs {
		t.Fatal("matching-cost hash should NOT need rehash")
	}
}
