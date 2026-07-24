package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifySignature verifies Meta's X-Hub-Signature-256 header against the raw
// request body using the app secret. The header is of the form
// "sha256=<hex>"; comparison is constant-time. Returns false on any mismatch,
// malformed header, or empty secret.
func VerifySignature(rawBody []byte, xHubSignature256Header, appSecret string) bool {
	if appSecret == "" || xHubSignature256Header == "" {
		return false
	}
	sig := strings.TrimPrefix(xHubSignature256Header, "sha256=")
	expected, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(rawBody)
	return hmac.Equal(expected, mac.Sum(nil))
}

// VerifySignature is also exposed as a Provider method for callers that already
// hold a configured Provider (uses cfg.AppSecret).
func (p *Provider) VerifySignature(rawBody []byte, xHubSignature256Header string) bool {
	return VerifySignature(rawBody, xHubSignature256Header, p.cfg.AppSecret)
}
