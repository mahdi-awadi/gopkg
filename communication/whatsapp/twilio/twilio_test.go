package twilio

import (
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestProvider_CodeAndChannels(t *testing.T) {
	p := New(Config{}, nil)
	if p.Code() != "twilio_whatsapp" {
		t.Fatalf("code=%q", p.Code())
	}
	chs := p.SupportedChannels()
	if len(chs) != 1 || chs[0] != provider.ChannelWhatsApp {
		t.Fatalf("expected [whatsapp], got %v", chs)
	}
}

func TestEnabledRequiresAllCreds(t *testing.T) {
	if New(Config{}, nil).Enabled() {
		t.Fatal("empty cfg should not be enabled")
	}
	if !New(Config{AccountSID: "a", AuthToken: "b", From: "+1"}, nil).Enabled() {
		t.Fatal("all creds present should enable")
	}
}

func TestWithWhatsAppPrefix(t *testing.T) {
	if withWhatsAppPrefix("+1234") != "whatsapp:+1234" {
		t.Fatal("prefix not added")
	}
	if withWhatsAppPrefix("whatsapp:+1234") != "whatsapp:+1234" {
		t.Fatal("prefix duplicated")
	}
}

func TestMaskPhone(t *testing.T) {
	if MaskPhone("whatsapp:+15551234") != "+1555****" {
		t.Fatalf("mask wrong: %s", MaskPhone("whatsapp:+15551234"))
	}
	if MaskPhone("+15551234") != "+1555****" {
		t.Fatalf("mask wrong: %s", MaskPhone("+15551234"))
	}
}
