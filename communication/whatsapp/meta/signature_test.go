package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_Valid(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)
	if !VerifySignature(body, sign(body, "APPSECRET"), "APPSECRET") {
		t.Fatal("expected valid signature to verify")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	body := []byte(`{"a":1}`)
	if VerifySignature(body, sign(body, "OTHER"), "APPSECRET") {
		t.Fatal("expected wrong-secret signature to fail")
	}
	if VerifySignature(body, "sha256=zzz", "APPSECRET") {
		t.Fatal("expected malformed hex to fail")
	}
	if VerifySignature(body, sign(body, "APPSECRET"), "") {
		t.Fatal("expected empty secret to fail")
	}
	if VerifySignature(body, "", "APPSECRET") {
		t.Fatal("expected empty header to fail")
	}
}

func TestProviderVerifySignature(t *testing.T) {
	body := []byte(`payload`)
	p := New(Config{PhoneNumberID: "p", AccessToken: "t", AppSecret: "S"}, nil)
	if !p.VerifySignature(body, sign(body, "S")) {
		t.Fatal("expected provider method to verify with cfg.AppSecret")
	}
}
