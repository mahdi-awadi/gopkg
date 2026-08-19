// Package cloudflare provides an email sender that delivers through a
// Cloudflare Worker HTTP endpoint. It implements
// communication/provider.EmailProvider, so it is drop-in compatible with the
// other email senders (sendgrid, ses).
//
// The Worker receives a JSON body (from, to, subject, text, html, optional
// attachments) and replies with a JSON object holding a message id. Requests
// carry the configured secret in an "Authorization: Bearer <token>" header.
//
// Construct via New; the returned *Provider is safe for concurrent use.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "cloudflare"

// Logger is the minimum logging contract this provider uses.
type Logger interface {
	Info(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

type noopLogger struct{}

func (noopLogger) Info(string, map[string]any)  {}
func (noopLogger) Warn(string, map[string]any)  {}
func (noopLogger) Error(string, map[string]any) {}

// Config holds the Cloudflare Worker endpoint and the default From identity.
// AuthToken is the shared secret sent as a Bearer token; never hardcode it.
type Config struct {
	WorkerURL string
	AuthToken string
	FromEmail string
	FromName  string
}

// Provider implements provider.EmailProvider via a Cloudflare Worker.
type Provider struct {
	cfg        Config
	httpClient *http.Client
	logger     Logger
}

// New constructs a Cloudflare Worker Provider. logger may be nil (becomes noop).
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	return &Provider{
		cfg:        cfg,
		httpClient: &http.Client{},
		logger:     logger,
	}
}

// Compile-time check.
var _ provider.EmailProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelEmail}
}

// ValidateConfig returns an error if required fields are missing.
func (p *Provider) ValidateConfig() error {
	if p.cfg.WorkerURL == "" {
		return fmt.Errorf("cloudflare: WorkerURL is required")
	}
	if p.cfg.AuthToken == "" {
		return fmt.Errorf("cloudflare: AuthToken is required")
	}
	if p.cfg.FromEmail == "" {
		return fmt.Errorf("cloudflare: FromEmail is required")
	}
	return nil
}

// Enabled returns true when WorkerURL, AuthToken and FromEmail are all present.
func (p *Provider) Enabled() bool {
	return p.cfg.WorkerURL != "" && p.cfg.AuthToken != "" && p.cfg.FromEmail != ""
}

// payload is the JSON body posted to the Worker.
type payload struct {
	From          string       `json:"from"`
	FromName      string       `json:"from_name,omitempty"`
	To            string       `json:"to"`
	Subject       string       `json:"subject"`
	Text          string       `json:"text,omitempty"`
	HTML          string       `json:"html,omitempty"`
	TemplateCode  string       `json:"template_code,omitempty"`
	CorrelationID string       `json:"correlation_id,omitempty"`
	Attachments   []attachment `json:"attachments,omitempty"`
}

// attachment is one base64-encoded file in the payload.
type attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     string `json:"content_base64"`
}

// result is the JSON the Worker returns on success.
type result struct {
	ID        string `json:"id"`
	MessageID string `json:"message_id"`
}

// Send sends a single email through the Worker.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	return p.send(ctx, req, nil)
}

// SendWithAttachments sends an email with file attachments through the Worker.
func (p *Provider) SendWithAttachments(ctx context.Context, req *provider.SendRequest, attachments []provider.Attachment) (*provider.SendResponse, error) {
	return p.send(ctx, req, attachments)
}

// GetStatus returns StatusSent; the Worker exposes no per-message status API.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusSent}, nil
}

func (p *Provider) send(ctx context.Context, req *provider.SendRequest, attachments []provider.Attachment) (*provider.SendResponse, error) {
	if req.RecipientEmail == "" {
		return p.fail("recipient email is required"), nil
	}
	if p.cfg.WorkerURL == "" {
		return p.fail("cloudflare: worker URL is required"), nil
	}

	body := payload{
		From:          p.cfg.FromEmail,
		FromName:      p.cfg.FromName,
		To:            req.RecipientEmail,
		Subject:       req.Subject,
		Text:          req.Body,
		HTML:          req.HTMLBody,
		TemplateCode:  req.TemplateCode,
		CorrelationID: req.CorrelationID,
	}
	for _, att := range attachments {
		body.Attachments = append(body.Attachments, attachment{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Content:     base64.StdEncoding.EncodeToString(att.Content),
		})
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return p.fail(fmt.Sprintf("cloudflare: encode payload: %v", err)), nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.WorkerURL, bytes.NewReader(raw))
	if err != nil {
		return p.fail(fmt.Sprintf("cloudflare: build request: %v", err)), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.cfg.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.cfg.AuthToken)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.logger.Error("cloudflare: send failed", map[string]any{
			"recipient": MaskEmail(req.RecipientEmail),
			"error":     err.Error(),
		})
		return p.fail(err.Error()), nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var res result
	_ = json.Unmarshal(respBody, &res)
	messageID := res.ID
	if messageID == "" {
		messageID = res.MessageID
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.logger.Warn("cloudflare: non-success status", map[string]any{
			"status_code": resp.StatusCode,
			"body":        string(respBody),
		})
		return &provider.SendResponse{
			Success:           false,
			ProviderCode:      ProviderCode,
			ProviderMessageID: messageID,
			Error:             fmt.Sprintf("cloudflare: status %d: %s", resp.StatusCode, string(respBody)),
			RawResponse:       respBody,
		}, nil
	}

	p.logger.Info("cloudflare: email sent", map[string]any{
		"recipient":  MaskEmail(req.RecipientEmail),
		"subject":    req.Subject,
		"message_id": messageID,
	})
	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: messageID,
		RawResponse:       respBody,
	}, nil
}

// fail builds a failed SendResponse carrying msg.
func (p *Provider) fail(msg string) *provider.SendResponse {
	return &provider.SendResponse{
		Success:      false,
		ProviderCode: ProviderCode,
		Error:        msg,
	}
}

// MaskEmail masks an email address for safe logging.
func MaskEmail(email string) string {
	if len(email) <= 4 {
		return "****"
	}
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "****"
	}
	if at <= 2 {
		return email[:1] + "***" + email[at:]
	}
	return email[:2] + "***" + email[at:]
}
