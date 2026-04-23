package twilio

import (
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestProvider_CodeAndChannels(t *testing.T) {
	p := New(Config{}, nil)
	if p.Code() != "twilio_voice" {
		t.Fatalf("code=%q", p.Code())
	}
	chs := p.SupportedChannels()
	if len(chs) != 1 || chs[0] != provider.ChannelVoice {
		t.Fatalf("expected [voice], got %v", chs)
	}
}

func TestEnabledRequiresAllCreds(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"empty", Config{}, false},
		{"sid only", Config{AccountSID: "a"}, false},
		{"all", Config{AccountSID: "a", AuthToken: "b", FromNumber: "+1"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := New(c.cfg, nil).Enabled(); got != c.want {
				t.Fatalf("Enabled=%v, want %v", got, c.want)
			}
		})
	}
}

func TestDefaultRejectURLApplied(t *testing.T) {
	p := New(Config{}, nil)
	if p.cfg.RejectURL != DefaultRejectURL {
		t.Fatalf("expected default reject URL applied, got %q", p.cfg.RejectURL)
	}
}

func TestMaskPhone(t *testing.T) {
	if MaskPhone("+15551234") != "+1555****" {
		t.Fatalf("mask wrong: %s", MaskPhone("+15551234"))
	}
	if MaskPhone("x") != "****" {
		t.Fatalf("short mask wrong: %s", MaskPhone("x"))
	}
}
