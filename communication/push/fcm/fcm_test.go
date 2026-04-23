package fcm

import (
	"testing"
)

func TestMaskToken(t *testing.T) {
	cases := map[string]string{
		"":         "****",
		"short":    "****",
		"12345678": "****",
		"123456789abc": "12345678****",
		"abcdefghijklmnop": "abcdefgh****",
	}
	for in, want := range cases {
		if got := MaskToken(in); got != want {
			t.Fatalf("MaskToken(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestProvider_ZeroValueNotEnabled(t *testing.T) {
	p := &Provider{}
	if p.Enabled() {
		t.Fatal("zero-value Provider should not be Enabled")
	}
}

func TestValidateConfig_RequiresProjectAndCreds(t *testing.T) {
	if err := (&Provider{}).ValidateConfig(); err == nil {
		t.Fatal("expected error on empty cfg")
	}
	if err := (&Provider{cfg: Config{ProjectID: "p"}}).ValidateConfig(); err == nil {
		t.Fatal("expected error when CredentialsJSON missing")
	}
}
