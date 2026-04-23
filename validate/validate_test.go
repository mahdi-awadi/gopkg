package validate

import (
	"errors"
	"testing"
)

func TestEmail(t *testing.T) {
	ok := []string{"a@b.co", "alice+tag@example.com", "user.name@sub.example.org"}
	bad := []string{"", "not-an-email", "@example.com", "a@"}
	for _, s := range ok {
		if err := Email(s); err != nil {
			t.Errorf("Email(%q) should pass, got %v", s, err)
		}
	}
	for _, s := range bad {
		if err := Email(s); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("Email(%q) should fail, got %v", s, err)
		}
	}
}

func TestPhoneE164(t *testing.T) {
	ok := []string{"+12025550199", "+447911123456", "+9647901234567"}
	bad := []string{"", "12025550199", "+0123", "+abc", "+", "+0"}
	for _, s := range ok {
		if err := PhoneE164(s); err != nil {
			t.Errorf("PhoneE164(%q) should pass, got %v", s, err)
		}
	}
	for _, s := range bad {
		if err := PhoneE164(s); !errors.Is(err, ErrInvalidPhone) {
			t.Errorf("PhoneE164(%q) should fail, got %v", s, err)
		}
	}
}

func TestURL(t *testing.T) {
	ok := []string{"http://example.com", "https://example.org/path?x=1", "ftp://host/f"}
	bad := []string{"", "not a url", "/relative/path", "http://"}
	for _, s := range ok {
		if err := URL(s); err != nil {
			t.Errorf("URL(%q) should pass, got %v", s, err)
		}
	}
	for _, s := range bad {
		if err := URL(s); !errors.Is(err, ErrInvalidURL) {
			t.Errorf("URL(%q) should fail, got %v", s, err)
		}
	}
}

func TestUUID(t *testing.T) {
	ok := []string{
		"018f4b5c-a29b-7aa1-b3cd-9e4a22e8f100", // v7
		"550e8400-e29b-41d4-a716-446655440000", // v4
	}
	bad := []string{"", "not-a-uuid", "ZZZ18f4b-a29b-7aa1-b3cd-9e4a22e8f100"}
	for _, s := range ok {
		if err := UUID(s); err != nil {
			t.Errorf("UUID(%q) should pass, got %v", s, err)
		}
	}
	for _, s := range bad {
		if err := UUID(s); !errors.Is(err, ErrInvalidUUID) {
			t.Errorf("UUID(%q) should fail, got %v", s, err)
		}
	}
}

func TestPassword_Defaults(t *testing.T) {
	p := DefaultPasswordPolicy()

	if err := Password("Short1!", p); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("expected too short, got %v", err)
	}
	if err := Password("alllowercase1!", p); !errors.Is(err, ErrPasswordNoUpper) {
		t.Errorf("expected no upper, got %v", err)
	}
	if err := Password("ALLUPPER1!", p); !errors.Is(err, ErrPasswordNoLower) {
		t.Errorf("expected no lower, got %v", err)
	}
	if err := Password("NoDigits!", p); !errors.Is(err, ErrPasswordNoDigit) {
		t.Errorf("expected no digit, got %v", err)
	}
	if err := Password("NoSymbols1", p); !errors.Is(err, ErrPasswordNoSymbol) {
		t.Errorf("expected no symbol, got %v", err)
	}
	if err := Password("Strong1!pw", p); err != nil {
		t.Errorf("expected pass, got %v", err)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if NormalizeEmail("  Alice@EXAMPLE.com  ") != "alice@example.com" {
		t.Fatal("normalize mismatch")
	}
}
