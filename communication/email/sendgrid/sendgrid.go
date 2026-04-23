// Package sendgrid provides a SendGrid implementation of
// communication/provider.EmailProvider.
//
// Construct via New; the returned *Provider is safe for concurrent use.
package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	sg "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "sendgrid"

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

// Config holds SendGrid credentials and the default From identity.
type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// Provider implements provider.EmailProvider via SendGrid.
type Provider struct {
	cfg    Config
	client *sg.Client
	logger Logger
}

// New constructs a SendGrid Provider. logger may be nil (becomes noop).
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	return &Provider{
		cfg:    cfg,
		client: sg.NewSendClient(cfg.APIKey),
		logger: logger,
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
	if p.cfg.APIKey == "" {
		return fmt.Errorf("sendgrid: APIKey is required")
	}
	if p.cfg.FromEmail == "" {
		return fmt.Errorf("sendgrid: FromEmail is required")
	}
	return nil
}

// Enabled returns true when APIKey and FromEmail are both present.
func (p *Provider) Enabled() bool {
	return p.cfg.APIKey != "" && p.cfg.FromEmail != ""
}

// Send sends a single email.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientEmail == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient email is required",
		}, nil
	}

	message := p.buildMessage(req)
	return p.sendMessage(ctx, message, req.RecipientEmail, req.Subject)
}

// SendWithAttachments sends an email with file attachments.
func (p *Provider) SendWithAttachments(ctx context.Context, req *provider.SendRequest, attachments []provider.Attachment) (*provider.SendResponse, error) {
	if req.RecipientEmail == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient email is required",
		}, nil
	}

	message := p.buildMessage(req)
	for _, att := range attachments {
		a := mail.NewAttachment()
		a.SetContent(string(att.Content))
		a.SetType(att.ContentType)
		a.SetFilename(att.Filename)
		a.SetDisposition("attachment")
		message.AddAttachment(a)
	}
	return p.sendMessage(ctx, message, req.RecipientEmail, req.Subject)
}

// GetStatus returns the delivery status for a previously-sent message.
// SendGrid doesn't expose a simple per-message status API; this returns
// StatusSent. For richer tracking, consume SendGrid Event Webhook externally.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{
		MessageID: messageID,
		Status:    provider.StatusSent,
	}, nil
}

func (p *Provider) buildMessage(req *provider.SendRequest) *mail.SGMailV3 {
	from := mail.NewEmail(p.cfg.FromName, p.cfg.FromEmail)
	to := mail.NewEmail("", req.RecipientEmail)
	m := mail.NewSingleEmail(from, req.Subject, to, req.Body, req.HTMLBody)
	if req.CorrelationID != "" {
		m.SetHeader("X-Correlation-ID", req.CorrelationID)
	}
	if req.TemplateCode != "" {
		m.SetHeader("X-Template-Code", req.TemplateCode)
	}
	return m
}

func (p *Provider) sendMessage(ctx context.Context, m *mail.SGMailV3, recipient, subject string) (*provider.SendResponse, error) {
	resp, err := p.client.SendWithContext(ctx, m)
	if err != nil {
		p.logger.Error("sendgrid: send failed", map[string]any{
			"recipient": MaskEmail(recipient),
			"error":     err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}

	raw, _ := json.Marshal(map[string]any{
		"status_code": resp.StatusCode,
		"body":        resp.Body,
		"headers":     resp.Headers,
	})

	var messageID string
	if v, ok := resp.Headers["X-Message-Id"]; ok && len(v) > 0 {
		messageID = v[0]
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		p.logger.Warn("sendgrid: non-success status", map[string]any{
			"status_code": resp.StatusCode,
			"body":        resp.Body,
		})
		return &provider.SendResponse{
			Success:           false,
			ProviderCode:      ProviderCode,
			ProviderMessageID: messageID,
			Error:             fmt.Sprintf("sendgrid: status %d: %s", resp.StatusCode, resp.Body),
			RawResponse:       raw,
		}, nil
	}

	p.logger.Info("sendgrid: email sent", map[string]any{
		"recipient":  MaskEmail(recipient),
		"subject":    subject,
		"message_id": messageID,
	})

	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: messageID,
		RawResponse:       raw,
	}, nil
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
