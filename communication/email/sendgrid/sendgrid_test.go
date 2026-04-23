package sendgrid

import (
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestNew_WithNilLogger(t *testing.T) {
	p := New(Config{APIKey: "SG.xxx", FromEmail: "from@example.com"}, nil)
	if p == nil {
		t.Fatal("nil Provider")
	}
	if !p.Enabled() {
		t.Fatal("expected Enabled=true")
	}
}

func TestProvider_EnabledRequiresKeyAndFrom(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"empty", Config{}, false},
		{"key only", Config{APIKey: "k"}, false},
		{"from only", Config{FromEmail: "f@x"}, false},
		{"both", Config{APIKey: "k", FromEmail: "f@x"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := New(c.cfg, nil)
			if p.Enabled() != c.want {
				t.Fatalf("got %v, want %v", p.Enabled(), c.want)
			}
		})
	}
}

func TestProvider_ValidateConfig(t *testing.T) {
	if err := New(Config{}, nil).ValidateConfig(); err == nil {
		t.Fatal("expected error on empty config")
	}
	if err := New(Config{APIKey: "k"}, nil).ValidateConfig(); err == nil {
		t.Fatal("expected error when FromEmail missing")
	}
	if err := New(Config{APIKey: "k", FromEmail: "f@x"}, nil).ValidateConfig(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestProvider_CodeAndChannels(t *testing.T) {
	p := New(Config{}, nil)
	if p.Code() != "sendgrid" {
		t.Fatalf("code=%q", p.Code())
	}
	chs := p.SupportedChannels()
	if len(chs) != 1 || chs[0] != provider.ChannelEmail {
		t.Fatalf("expected [email], got %v", chs)
	}
}

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
