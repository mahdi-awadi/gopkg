// Package adminli provides a WhatsApp-via-Adminli implementation of
// communication/provider.WhatsAppProvider. It talks directly to Adminli's
// Programmable Messaging API (https://api.adminli.app) — POST /v1/messages with
// a per-tenant Bearer API key, which alone identifies the sending WhatsApp
// Business number (no from-number is configured or sent).
//
// Adminli currently accepts approved-template sends only; free-form session
// text and media are on Adminli's backlog and return an unsupported failure.
package adminli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

const (
	// ProviderCode identifies this provider in the registry and logs.
	ProviderCode = "adminli_whatsapp"
	// ModuleCode is the partner module-binding code.
	ModuleCode = "ADMINLI_WHATSAPP"
	// DefaultBaseURL is the production host used when Config.BaseURL is empty.
	DefaultBaseURL = "https://api.adminli.app"
)

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

// Config holds a single Adminli tenant's messaging credentials. The API key
// resolves to that tenant's WhatsApp Business number.
type Config struct {
	// APIKey is the tenant's Adminli API key, sent as a Bearer token.
	APIKey string
	// BaseURL overrides DefaultBaseURL (useful for tests). Empty means production.
	BaseURL string
	// Timeout for individual HTTP requests. Zero means 15 seconds.
	Timeout time.Duration
}

// Provider sends WhatsApp messages via Adminli for one tenant (Config.APIKey).
type Provider struct {
	cfg    Config
	client *http.Client
	logger Logger
}

// New constructs an Adminli WhatsApp Provider. logger may be nil.
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &Provider{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}, logger: logger}
}

// Compile-time check.
var _ provider.WhatsAppProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelWhatsApp}
}

// ValidateConfig returns an error when the API key is missing.
func (p *Provider) ValidateConfig() error {
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return fmt.Errorf("adminli: APIKey is required")
	}
	return nil
}

// Enabled reports whether the API key is set.
func (p *Provider) Enabled() bool { return strings.TrimSpace(p.cfg.APIKey) != "" }

// Send: Adminli accepts template sends only; free-form session text is not supported.
func (p *Provider) Send(_ context.Context, _ *provider.SendRequest) (*provider.SendResponse, error) {
	return failResp("adminli: free-form session send is not supported; use SendTemplate"), nil
}

// SendTemplate sends an approved WhatsApp template with positional body
// parameters. For button parameters or explicit language, use SendTemplateMessage.
func (p *Provider) SendTemplate(ctx context.Context, req *provider.SendRequest, templateName string, parameters []string) (*provider.SendResponse, error) {
	if req == nil || req.RecipientPhone == "" {
		return failResp("adminli: recipient phone is required"), nil
	}
	if templateName == "" {
		return failResp("adminli: template name is required"), nil
	}
	lang := ""
	if req != nil {
		lang = req.Language
	}
	res := p.SendTemplateMessage(ctx, req.RecipientPhone, TemplateSend{
		Name:           templateName,
		Language:       lang,
		BodyParameters: parameters,
	})
	return res.toSendResponse(), nil
}

// SendMedia: Adminli's public API has no media send yet.
func (p *Provider) SendMedia(context.Context, *provider.SendRequest, string, string) (*provider.SendResponse, error) {
	return failResp("adminli: media send is not supported"), nil
}

// GetStatus returns StatusSent. Adminli exposes no message-status read API or
// delivery webhook yet, so anything more specific would be fabricated.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusSent}, nil
}

// Result is the outcome of a native SendTemplateMessage call. It carries the
// retryability + HTTP status that the thin provider.SendResponse cannot.
type Result struct {
	Success   bool
	MessageID string
	Error     string
	Retryable bool
	Status    int
	Raw       []byte
}

func (r *Result) toSendResponse() *provider.SendResponse {
	return &provider.SendResponse{
		Success:           r.Success,
		ProviderCode:      ProviderCode,
		ProviderMessageID: r.MessageID,
		Error:             r.Error,
		RawResponse:       r.Raw,
	}
}

// SendTemplateMessage is the full-fidelity native send: a recipient phone
// (normalized to E.164) plus an approved template with optional body and button
// parameters and explicit language. Per-message failures are returned in Result
// (Success=false, with Retryable/Status), never as a Go error.
func (p *Provider) SendTemplateMessage(ctx context.Context, to string, tmpl TemplateSend) *Result {
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return &Result{Error: "adminli: api_key not configured"}
	}
	if strings.TrimSpace(tmpl.Name) == "" {
		return &Result{Error: "adminli: template name is required"}
	}

	payload := SendMessageRequest{To: NormalizePhone(to), Template: &tmpl}
	raw, err := json.Marshal(payload)
	if err != nil {
		return &Result{Error: fmt.Sprintf("adminli: marshal request: %v", err)}
	}

	url := strings.TrimSuffix(p.cfg.BaseURL, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return &Result{Error: fmt.Sprintf("adminli: create request: %v", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.logger.Error("adminli send failed", map[string]any{
			"recipient": MaskPhone(to), "template": tmpl.Name, "error": err.Error(),
		})
		return &Result{Error: fmt.Sprintf("adminli: http call: %v", err), Retryable: true}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Result{Error: fmt.Sprintf("adminli: read response: %v", err), Retryable: true}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		_ = json.Unmarshal(respBody, &apiErr)
		message := apiErr.Reason()
		if message == "" {
			message = strings.TrimSpace(string(respBody))
		}
		p.logger.Error("adminli send rejected", map[string]any{
			"recipient": MaskPhone(to), "template": tmpl.Name,
			"status": resp.StatusCode, "error": message,
		})
		return &Result{
			Error:     fmt.Sprintf("adminli: http %d: %s", resp.StatusCode, message),
			Retryable: isRetryableStatus(resp.StatusCode),
			Status:    resp.StatusCode,
			Raw:       respBody,
		}
	}

	var decoded SendMessageResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return &Result{Error: fmt.Sprintf("adminli: decode response: %v", err), Status: resp.StatusCode, Raw: respBody}
	}

	p.logger.Info("WhatsApp message sent via Adminli", map[string]any{
		"recipient": MaskPhone(to), "template": tmpl.Name, "message_id": decoded.MessageID,
	})
	return &Result{Success: true, MessageID: decoded.MessageID, Status: resp.StatusCode, Raw: respBody}
}

func failResp(msg string) *provider.SendResponse {
	return &provider.SendResponse{Success: false, ProviderCode: ProviderCode, Error: msg}
}

// isRetryableStatus reports whether re-sending could plausibly succeed.
func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// NormalizePhone strips any "whatsapp:" prefix and ensures a leading "+".
// Adminli defaults bare numbers to Iraq, so an explicit E.164 value is sent.
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(strings.TrimPrefix(phone, "whatsapp:"))
	if phone != "" && !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	return phone
}

// MaskPhone returns a partially masked number safe for logging.
func MaskPhone(phone string) string {
	phone = strings.TrimPrefix(phone, "whatsapp:")
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:len(phone)-4] + "****"
}
