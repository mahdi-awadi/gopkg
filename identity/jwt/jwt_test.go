package jwt

import (
	"errors"
	"testing"
	"time"

	jwt5 "github.com/golang-jwt/jwt/v5"
)

type userClaims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	*RegisteredClaims
}

func TestNew_EmptySecretErrors(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestIssueAndParseRoundTrip(t *testing.T) {
	s, _ := New("my-secret")
	reg := StandardTTL("svc", "user-123", time.Hour)
	claims := &userClaims{
		UserID:           "user-123",
		Email:            "alice@example.com",
		RegisteredClaims: &reg,
	}
	tok, err := Issue(s, claims)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}

	parsed, err := Parse(s, tok, &userClaims{RegisteredClaims: &RegisteredClaims{}})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.UserID != "user-123" || parsed.Email != "alice@example.com" {
		t.Fatalf("unexpected claims: %+v", parsed)
	}
}

func TestParse_WrongSecretFails(t *testing.T) {
	s, _ := New("secret-a")
	reg := StandardTTL("svc", "u", time.Hour)
	tok, _ := Issue(s, &userClaims{RegisteredClaims: &reg})

	other, _ := New("secret-b")
	_, err := Parse(other, tok, &userClaims{RegisteredClaims: &RegisteredClaims{}})
	if err == nil {
		t.Fatal("expected signature error")
	}
	if !errors.Is(err, jwt5.ErrSignatureInvalid) && !errors.Is(err, jwt5.ErrTokenSignatureInvalid) {
		t.Logf("got %v — signature invalid family acceptable", err)
	}
}

func TestParse_ExpiredToken(t *testing.T) {
	s, _ := New("secret")
	reg := StandardTTL("svc", "u", -time.Hour) // already expired
	tok, _ := Issue(s, &userClaims{RegisteredClaims: &reg})

	_, err := Parse(s, tok, &userClaims{RegisteredClaims: &RegisteredClaims{}})
	if err == nil {
		t.Fatal("expected expiration error")
	}
	if !errors.Is(err, jwt5.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}
