package adminli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestSendTemplateMessage_Success(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"messageId":"wamid.123","to":"+9647701234567","channel":"whatsapp","status":"sent"}`)
	}))
	defer srv.Close()

	p := New(Config{APIKey: "ali_live_test", BaseURL: srv.URL}, nil)
	res := p.SendTemplateMessage(context.Background(), "9647701234567", TemplateSend{
		Name: "otp", Language: "ar", BodyParameters: []string{"123456"},
	})

	if !res.Success || res.MessageID != "wamid.123" {
		t.Fatalf("expected success wamid.123, got success=%v id=%q err=%q", res.Success, res.MessageID, res.Error)
	}
	if gotAuth != "Bearer ali_live_test" {
		t.Errorf("expected Bearer auth, got %q", gotAuth)
	}
	// Phone must be normalized to E.164 with a leading +.
	var sent SendMessageRequest
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("bad request body: %v", err)
	}
	if sent.To != "+9647701234567" {
		t.Errorf("expected normalized +E.164, got %q", sent.To)
	}
	if sent.Template == nil || sent.Template.Name != "otp" || sent.Template.Language != "ar" {
		t.Errorf("template not sent correctly: %+v", sent.Template)
	}
}

func TestSendTemplateMessage_ErrorNotRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"invalid_api_key"}`)
	}))
	defer srv.Close()

	res := New(Config{APIKey: "bad", BaseURL: srv.URL}, nil).
		SendTemplateMessage(context.Background(), "+9647701234567", TemplateSend{Name: "otp"})

	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Retryable {
		t.Error("401 must not be retryable")
	}
	if res.Status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", res.Status)
	}
	if want := "invalid_api_key"; !contains(res.Error, want) {
		t.Errorf("expected error to mention %q, got %q", want, res.Error)
	}
}

func TestSend_FreeFormUnsupported(t *testing.T) {
	res, _ := New(Config{APIKey: "x"}, nil).Send(context.Background(), &provider.SendRequest{RecipientPhone: "+964770"})
	if res.Success {
		t.Fatal("free-form Send must fail")
	}
}

func TestValidateConfig(t *testing.T) {
	if err := New(Config{}, nil).ValidateConfig(); err == nil {
		t.Error("empty APIKey should be invalid")
	}
	if err := New(Config{APIKey: "x"}, nil).ValidateConfig(); err != nil {
		t.Errorf("valid config errored: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
