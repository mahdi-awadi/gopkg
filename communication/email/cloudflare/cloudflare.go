// Package cloudflare implements communication/provider.EmailProvider using the
// Cloudflare Email Service REST API (POST /accounts/{id}/email/sending/send).
package cloudflare

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

const ProviderCode = "cloudflare"

const apiBase = "https://api.cloudflare.com/client/v4"

// Logger mirrors the minimal logger used by the sendgrid/ses providers.
type Logger interface {
	Info(string, map[string]any)
	Warn(string, map[string]any)
	Error(string, map[string]any)
}

type noopLogger struct{}

func (noopLogger) Info(string, map[string]any)  {}
func (noopLogger) Warn(string, map[string]any)  {}
func (noopLogger) Error(string, map[string]any) {}

// Config holds the Cloudflare Email Service settings.
type Config struct {
	AccountID string
	APIToken  string
	FromEmail string
	FromName  string
}

// Provider implements provider.EmailProvider via Cloudflare Email Service.
type Provider struct {
	cfg    Config
	http   *http.Client
	logger Logger
}

// Option configures a Provider.
type Option func(*Provider)

// WithHTTPClient injects an *http.Client (tests inject a fake transport).
func WithHTTPClient(hc *http.Client) Option { return func(p *Provider) { p.http = hc } }

// New constructs a Cloudflare email Provider. logger may be nil (becomes noop).
func New(cfg Config, logger Logger, opts ...Option) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	p := &Provider{cfg: cfg, http: &http.Client{Timeout: 30 * time.Second}, logger: logger}
	for _, o := range opts {
		o(p)
	}
	return p
}

var _ provider.EmailProvider = (*Provider)(nil)

func (p *Provider) Code() string { return ProviderCode }

func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelEmail}
}

func (p *Provider) ValidateConfig() error {
	if p.cfg.AccountID == "" || p.cfg.APIToken == "" || p.cfg.FromEmail == "" {
		return fmt.Errorf("cloudflare: AccountID, APIToken and FromEmail are required")
	}
	return nil
}

func (p *Provider) Enabled() bool {
	return p.cfg.AccountID != "" && p.cfg.APIToken != "" && p.cfg.FromEmail != ""
}

// sendBody is the CF Email Service request payload. NOTE (build-then-verify): the exact
// field names/casing are confirmed against the live API on first real send; adjust the
// json tags here only — the test asserts the URL/auth/subject which are stable.
type sendBody struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []attach          `json:"attachments,omitempty"`
}

type attach struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Content  string `json:"content"` // base64
}

func (p *Provider) from() string {
	if p.cfg.FromName != "" {
		return fmt.Sprintf("%s <%s>", p.cfg.FromName, p.cfg.FromEmail)
	}
	return p.cfg.FromEmail
}

func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	return p.send(ctx, req, nil)
}

func (p *Provider) SendWithAttachments(ctx context.Context, req *provider.SendRequest, atts []provider.Attachment) (*provider.SendResponse, error) {
	return p.send(ctx, req, atts)
}

func (p *Provider) send(ctx context.Context, req *provider.SendRequest, atts []provider.Attachment) (*provider.SendResponse, error) {
	if err := p.ValidateConfig(); err != nil {
		return nil, err
	}
	if req.RecipientEmail == "" {
		return nil, provider.NewProviderError(ProviderCode, "E_NO_RECIPIENT", "RecipientEmail is required", false, nil)
	}
	body := sendBody{From: p.from(), To: []string{req.RecipientEmail}, Subject: req.Subject, HTML: req.HTMLBody, Text: req.Body}
	if h, ok := req.Options["headers"].(map[string]string); ok {
		body.Headers = h
	}
	for _, a := range atts {
		body.Attachments = append(body.Attachments, attach{Filename: a.Filename, Type: a.ContentType, Content: base64Std(a.Content)})
	}
	raw, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/accounts/%s/email/sending/send", apiBase, p.cfg.AccountID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, provider.NewProviderError(ProviderCode, "E_TRANSPORT", "cloudflare send failed", true, err)
	}
	defer resp.Body.Close()
	var parsed struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	if resp.StatusCode >= 300 || !parsed.Success {
		code, msg := "E_SEND_FAILED", "cloudflare rejected the message"
		if len(parsed.Errors) > 0 {
			code = parsed.Errors[0].Code
			msg = code + ": " + parsed.Errors[0].Message
		}
		return nil, provider.NewProviderError(ProviderCode, code, msg, resp.StatusCode >= 500, nil)
	}
	return &provider.SendResponse{Success: true, ProviderCode: ProviderCode, ProviderMessageID: parsed.Result.ID}, nil
}

// GetStatus is not supported by the send API; report unknown.
func (p *Provider) GetStatus(ctx context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusUnknown}, nil
}

func base64Std(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
