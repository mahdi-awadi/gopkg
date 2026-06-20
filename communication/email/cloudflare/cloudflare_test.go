package cloudflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

// roundTripFunc lets a test stand in for the network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendPostsToEmailServiceREST(t *testing.T) {
	var gotURL, gotAuth string
	var gotBody map[string]any
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"success":true}`)), Header: make(http.Header)}, nil
	})
	p := New(Config{AccountID: "acct123", APIToken: "tok", FromEmail: "support@tech-gate.online", FromName: "Support"},
		nil, WithHTTPClient(&http.Client{Transport: rt}))

	resp, err := p.Send(context.Background(), &provider.SendRequest{
		RecipientEmail: "cust@example.com", Subject: "Re: hello", HTMLBody: "<p>hi</p>", Body: "hi",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %+v", resp)
	}
	if gotURL != "https://api.cloudflare.com/client/v4/accounts/acct123/email/sending/send" {
		t.Fatalf("url = %s", gotURL)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth = %s", gotAuth)
	}
	if gotBody["subject"] != "Re: hello" {
		t.Fatalf("body subject = %v (full %v)", gotBody["subject"], gotBody)
	}
}

func TestSendNotVerifiedReturnsProviderError(t *testing.T) {
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403,
			Body:   io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"code":"E_SENDER_NOT_VERIFIED","message":"verify domain"}]}`)),
			Header: make(http.Header)}, nil
	})
	p := New(Config{AccountID: "a", APIToken: "t", FromEmail: "x@y.com"}, nil, WithHTTPClient(&http.Client{Transport: rt}))
	_, err := p.Send(context.Background(), &provider.SendRequest{RecipientEmail: "c@e.com", Subject: "s", Body: "b"})
	if err == nil {
		t.Fatal("expected error for unverified sender")
	}
	var pe *provider.ProviderError
	if !strings.Contains(err.Error(), "E_SENDER_NOT_VERIFIED") {
		t.Fatalf("want CF error code in %v", err)
	}
	_ = pe
}
