package cloudflare

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

// TestProvider_Send drives Send through a stub worker endpoint (httptest) and
// covers the three outcomes: success, non-2xx, and bad config.
func TestProvider_Send(t *testing.T) {
	cases := []struct {
		name        string
		badConfig   bool   // when true, no server is started and WorkerURL stays empty
		respStatus  int    // status the stub worker returns
		respBody    string // body the stub worker returns
		wantSuccess bool
		wantMsgID   string
		wantErrSub  string // substring that must appear in resp.Error
	}{
		{
			name:        "success",
			respStatus:  http.StatusOK,
			respBody:    `{"id":"msg_123"}`,
			wantSuccess: true,
			wantMsgID:   "msg_123",
		},
		{
			name:       "non-2xx is a failure with a useful message",
			respStatus: http.StatusInternalServerError,
			respBody:   "worker exploded",
			wantErrSub: "500",
		},
		{
			name:       "bad config: empty worker URL",
			badConfig:  true,
			wantErrSub: "worker URL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{AuthToken: "secret", FromEmail: "from@example.com", FromName: "Example"}

			if !tc.badConfig {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify the request the worker receives.
					if r.Method != http.MethodPost {
						t.Errorf("method = %q, want POST", r.Method)
					}
					if got := r.Header.Get("Authorization"); got != "Bearer secret" {
						t.Errorf("Authorization = %q, want %q", got, "Bearer secret")
					}
					if got := r.Header.Get("Content-Type"); got != "application/json" {
						t.Errorf("Content-Type = %q, want application/json", got)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode body: %v", err)
					}
					if body["to"] != "bob@example.com" {
						t.Errorf("body.to = %v, want bob@example.com", body["to"])
					}
					if body["from"] != "from@example.com" {
						t.Errorf("body.from = %v, want from@example.com", body["from"])
					}
					if body["subject"] != "Hi" {
						t.Errorf("body.subject = %v, want Hi", body["subject"])
					}
					w.WriteHeader(tc.respStatus)
					_, _ = w.Write([]byte(tc.respBody))
				}))
				defer srv.Close()
				cfg.WorkerURL = srv.URL
			}

			p := New(cfg, nil)
			resp, err := p.Send(context.Background(), &provider.SendRequest{
				RecipientEmail: "bob@example.com",
				Subject:        "Hi",
				Body:           "Plain text body",
				HTMLBody:       "<p>HTML body</p>",
			})
			if err != nil {
				t.Fatalf("unexpected go error: %v", err)
			}
			if resp.Success != tc.wantSuccess {
				t.Fatalf("Success = %v, want %v (error=%q)", resp.Success, tc.wantSuccess, resp.Error)
			}
			if resp.ProviderCode != ProviderCode {
				t.Errorf("ProviderCode = %q, want %q", resp.ProviderCode, ProviderCode)
			}
			if tc.wantMsgID != "" && resp.ProviderMessageID != tc.wantMsgID {
				t.Errorf("ProviderMessageID = %q, want %q", resp.ProviderMessageID, tc.wantMsgID)
			}
			if tc.wantErrSub != "" && !strings.Contains(resp.Error, tc.wantErrSub) {
				t.Errorf("Error = %q, want it to contain %q", resp.Error, tc.wantErrSub)
			}
		})
	}
}

// TestProvider_SendWithAttachments checks that attachments reach the worker
// base64-encoded in the JSON payload.
func TestProvider_SendWithAttachments(t *testing.T) {
	var gotAttach []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if raw, ok := body["attachments"].([]any); ok {
			for _, a := range raw {
				gotAttach = append(gotAttach, a.(map[string]any))
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"m1"}`))
	}))
	defer srv.Close()

	p := New(Config{WorkerURL: srv.URL, AuthToken: "t", FromEmail: "from@example.com"}, nil)
	resp, err := p.SendWithAttachments(context.Background(),
		&provider.SendRequest{RecipientEmail: "bob@example.com", Subject: "Invoice"},
		[]provider.Attachment{{Filename: "invoice.pdf", ContentType: "application/pdf", Content: []byte("PDFDATA")}},
	)
	if err != nil {
		t.Fatalf("unexpected go error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("Success = false, want true (error=%q)", resp.Error)
	}
	if len(gotAttach) != 1 {
		t.Fatalf("worker got %d attachments, want 1", len(gotAttach))
	}
	if gotAttach[0]["filename"] != "invoice.pdf" {
		t.Errorf("filename = %v, want invoice.pdf", gotAttach[0]["filename"])
	}
	wantB64 := base64.StdEncoding.EncodeToString([]byte("PDFDATA"))
	if gotAttach[0]["content_base64"] != wantB64 {
		t.Errorf("content_base64 = %v, want %v", gotAttach[0]["content_base64"], wantB64)
	}
}

func TestProvider_EnabledRequiresURLTokenAndFrom(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"empty", Config{}, false},
		{"url only", Config{WorkerURL: "https://w"}, false},
		{"url+token", Config{WorkerURL: "https://w", AuthToken: "t"}, false},
		{"all", Config{WorkerURL: "https://w", AuthToken: "t", FromEmail: "f@x"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := New(c.cfg, nil).Enabled(); got != c.want {
				t.Fatalf("Enabled() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestProvider_ValidateConfig(t *testing.T) {
	if err := New(Config{}, nil).ValidateConfig(); err == nil {
		t.Fatal("expected error on empty config")
	}
	if err := New(Config{WorkerURL: "https://w", AuthToken: "t"}, nil).ValidateConfig(); err == nil {
		t.Fatal("expected error when FromEmail missing")
	}
	if err := New(Config{WorkerURL: "https://w", AuthToken: "t", FromEmail: "f@x"}, nil).ValidateConfig(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestProvider_CodeAndChannels(t *testing.T) {
	p := New(Config{}, nil)
	if p.Code() != "cloudflare" {
		t.Fatalf("code = %q", p.Code())
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
			t.Fatalf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
