package meta

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestCodeAndChannels(t *testing.T) {
	p := New(Config{}, nil)
	if p.Code() != "meta_whatsapp" {
		t.Fatalf("code=%q", p.Code())
	}
	chs := p.SupportedChannels()
	if len(chs) != 1 || chs[0] != provider.ChannelWhatsApp {
		t.Fatalf("expected [whatsapp], got %v", chs)
	}
}

func TestValidateConfig(t *testing.T) {
	if err := (&Provider{cfg: Config{}}).ValidateConfig(); err == nil {
		t.Fatal("expected error on empty cfg")
	}
	if err := (&Provider{cfg: Config{PhoneNumberID: "p"}}).ValidateConfig(); err == nil {
		t.Fatal("expected error without token")
	}
	if err := (&Provider{cfg: Config{PhoneNumberID: "p", AccessToken: "t"}}).ValidateConfig(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestEnabled(t *testing.T) {
	if New(Config{}, nil).Enabled() {
		t.Fatal("empty cfg should not be enabled")
	}
	if !New(Config{PhoneNumberID: "p", AccessToken: "t"}, nil).Enabled() {
		t.Fatal("valid cfg should be enabled")
	}
}

func TestStripPhonePrefix(t *testing.T) {
	cases := map[string]string{
		"+1234":         "1234",
		"whatsapp:+5":   "5",
		"whatsapp:5":    "5",
		"5":             "5",
	}
	for in, want := range cases {
		if got := stripPhonePrefix(in); got != want {
			t.Fatalf("stripPhonePrefix(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestMaskPhone(t *testing.T) {
	if MaskPhone("+15551234") != "1555****" {
		t.Fatalf("mask wrong: %s", MaskPhone("+15551234"))
	}
	if MaskPhone("whatsapp:+15551234") != "1555****" {
		t.Fatalf("mask wrong: %s", MaskPhone("whatsapp:+15551234"))
	}
}

func TestSend_HappyPathAgainstFakeServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"type":"text"`) {
			t.Errorf("expected type:text in body, got %s", string(body))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","messages":[{"id":"wamid.ABC"}]}`))
	}))
	defer srv.Close()

	p := New(Config{
		PhoneNumberID: "1234567890",
		AccessToken:   "TOKEN",
		GraphAPIBase:  srv.URL,
	}, nil)
	resp, err := p.Send(context.Background(), &provider.SendRequest{
		RecipientPhone: "+1234",
		Body:           "Hi",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !resp.Success || resp.ProviderMessageID != "wamid.ABC" {
		t.Fatalf("got %+v", resp)
	}
}

func TestSend_APIErrorReturnsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(MessagesResponse{
			Error: &APIError{Code: 190, Message: "Token expired", Type: "OAuthException"},
		})
	}))
	defer srv.Close()

	p := New(Config{PhoneNumberID: "p", AccessToken: "t", GraphAPIBase: srv.URL}, nil)
	resp, _ := p.Send(context.Background(), &provider.SendRequest{RecipientPhone: "+1", Body: "hi"})
	if resp.Success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(resp.Error, "190") {
		t.Fatalf("expected error code 190, got %q", resp.Error)
	}
}
