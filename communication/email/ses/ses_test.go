package ses

import (
	"testing"
)

func TestMaskEmail(t *testing.T) {
	cases := map[string]string{
		"":                  "****",
		"a@b":               "****",
		"ab@c.com":          "a***@c.com",
		"alice@example.com": "al***@example.com",
		"a@example.com":     "a***@example.com",
	}
	for in, want := range cases {
		if got := MaskEmail(in); got != want {
			t.Fatalf("MaskEmail(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestProvider_ZeroValueNotEnabled(t *testing.T) {
	p := &Provider{}
	if p.Enabled() {
		t.Fatal("zero-value Provider should not be Enabled")
	}
}
