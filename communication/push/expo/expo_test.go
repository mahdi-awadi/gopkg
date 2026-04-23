package expo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestIsExpoToken(t *testing.T) {
	cases := map[string]bool{
		"ExponentPushToken[abc]": true,
		"ExpoPushToken[xyz]":     true,
		"fcm-token":              false,
		"":                       false,
	}
	for in, want := range cases {
		if got := IsExpoToken(in); got != want {
			t.Fatalf("IsExpoToken(%q)=%v, want %v", in, got, want)
		}
	}
}

func TestProvider_Enabled(t *testing.T) {
	p := New(nil)
	if !p.Enabled() {
		t.Fatal("expected Enabled=true for Expo (no creds)")
	}
}

func TestProvider_CodeAndChannels(t *testing.T) {
	p := New(nil)
	if p.Code() != "expo_push" {
		t.Fatalf("code=%q", p.Code())
	}
	chs := p.SupportedChannels()
	if len(chs) != 1 || chs[0] != provider.ChannelPush {
		t.Fatalf("expected [push], got %v", chs)
	}
}

func TestSend_NonExpoTokenFails(t *testing.T) {
	p := New(nil)
	resp, err := p.Send(context.Background(), &provider.SendRequest{
		RecipientDeviceToken: "fcm-style-token",
		Body:                 "hi",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Success || !strings.Contains(resp.Error, "not an Expo") {
		t.Fatalf("expected non-expo error, got %+v", resp)
	}
}

func TestSend_HappyPathAgainstFakeServer(t *testing.T) {
	// Fake Expo server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"data":[{"id":"ticket-1","status":"ok"}]}`))
	}))
	defer srv.Close()

	p := New(nil)
	p.endpoint = srv.URL

	resp, err := p.Send(context.Background(), &provider.SendRequest{
		RecipientDeviceToken: "ExponentPushToken[abcd1234]",
		Subject:              "Hello",
		Body:                 "Body",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !resp.Success || resp.ProviderMessageID != "ticket-1" {
		t.Fatalf("got %+v, want success with ticket-1", resp)
	}
}

func TestMaskToken(t *testing.T) {
	if MaskToken("ExponentPushToken[abc]") != "Exponent****" {
		t.Fatalf("unexpected mask: %s", MaskToken("ExponentPushToken[abc]"))
	}
	if MaskToken("short") != "****" {
		t.Fatalf("unexpected mask for short: %s", MaskToken("short"))
	}
}
