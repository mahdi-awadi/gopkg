package twilio

import (
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestNew_WithNilLoggerReturnsWorkingProvider(t *testing.T) {
	p := New(Config{AccountSID: "a", AuthToken: "b", FromNumber: "+1"}, nil)
	if p == nil {
		t.Fatal("New returned nil")
	}
	if !p.Enabled() {
		t.Fatal("expected Enabled=true with all creds")
	}
}

func TestProvider_EnabledRequiresAllCreds(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"empty", Config{}, false},
		{"sid only", Config{AccountSID: "a"}, false},
		{"no from", Config{AccountSID: "a", AuthToken: "b"}, false},
		{"all", Config{AccountSID: "a", AuthToken: "b", FromNumber: "+1"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := New(c.cfg, nil)
			if p.Enabled() != c.want {
				t.Fatalf("Enabled=%v, want %v", p.Enabled(), c.want)
			}
		})
	}
}

func TestProvider_Code(t *testing.T) {
	p := New(Config{}, nil)
	if p.Code() != "twilio_sms" {
		t.Fatalf("got %q, want twilio_sms", p.Code())
	}
}

func TestProvider_SupportedChannelsIsSMS(t *testing.T) {
	p := New(Config{}, nil)
	chs := p.SupportedChannels()
	if len(chs) != 1 || chs[0] != provider.ChannelSMS {
		t.Fatalf("expected [SMS], got %v", chs)
	}
}

func TestCostEstimate_KnownCountryReturnsDefaultPricing(t *testing.T) {
	p := New(Config{}, nil)
	price, cur, err := p.CostEstimate(nil, "+19175550100")
	if err != nil || cur != "USD" || price == 0 {
		t.Fatalf("got %v %q %v, expected non-zero USD price", price, cur, err)
	}
}

func TestCostEstimate_UnknownCountryReturnsFallback(t *testing.T) {
	p := New(Config{}, nil)
	price, cur, err := p.CostEstimate(nil, "+99912345")
	if err != nil || cur != "USD" || price != 0.05 {
		t.Fatalf("got %v %q %v, want 0.05 USD", price, cur, err)
	}
}

func TestMaskPhone(t *testing.T) {
	cases := map[string]string{
		"":              "****",
		"1":             "****",
		"12":            "****",
		"1234":          "****",
		"+15551234":     "+1555****",
		"+19175550100":  "+1917555****",
	}
	for in, want := range cases {
		if got := MaskPhone(in); got != want {
			t.Fatalf("MaskPhone(%q)=%q, want %q", in, got, want)
		}
	}
}
