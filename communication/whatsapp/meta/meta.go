// Package meta provides a WhatsApp-via-Meta-Cloud-API implementation of
// communication/provider.WhatsAppProvider. Talks directly to
// graph.facebook.com — no third-party SDK.
package meta

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

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "meta_whatsapp"

// DefaultGraphAPIBase is the public Meta Graph API base URL.
const DefaultGraphAPIBase = "https://graph.facebook.com/v21.0"

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

// Config holds Meta WhatsApp Cloud API credentials.
type Config struct {
	// PhoneNumberID is the WhatsApp Business phone number ID (not the E.164 number).
	PhoneNumberID string
	// AccessToken is the permanent or system-user access token.
	AccessToken string
	// GraphAPIBase overrides the default graph.facebook.com URL (useful for tests).
	GraphAPIBase string
	// Timeout for individual HTTP requests. Zero means 15 seconds.
	Timeout time.Duration
}

// Provider implements provider.WhatsAppProvider via Meta Cloud API.
type Provider struct {
	cfg    Config
	client *http.Client
	logger Logger
}

// New constructs a Meta WhatsApp Provider. logger may be nil.
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	if cfg.GraphAPIBase == "" {
		cfg.GraphAPIBase = DefaultGraphAPIBase
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &Provider{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		logger: logger,
	}
}

// Compile-time check.
var _ provider.WhatsAppProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelWhatsApp}
}

// ValidateConfig returns an error if PhoneNumberID or AccessToken is missing.
func (p *Provider) ValidateConfig() error {
	if p.cfg.PhoneNumberID == "" {
		return fmt.Errorf("meta: PhoneNumberID is required")
	}
	if p.cfg.AccessToken == "" {
		return fmt.Errorf("meta: AccessToken is required")
	}
	return nil
}

// Enabled returns true when PhoneNumberID and AccessToken are set.
func (p *Provider) Enabled() bool {
	return p.cfg.PhoneNumberID != "" && p.cfg.AccessToken != ""
}

// MessagesResponse is the envelope returned by /messages.
type MessagesResponse struct {
	MessagingProduct string `json:"messaging_product,omitempty"`
	Messages         []struct {
		ID string `json:"id"`
	} `json:"messages,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

// APIError represents a Meta Graph API error envelope.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

// Send sends a WhatsApp text message.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}
	to := stripPhonePrefix(req.RecipientPhone)
	text := req.Body
	if req.HTMLBody != "" {
		text = req.HTMLBody
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]any{
			"preview_url": false,
			"body":        text,
		},
	}
	return p.postMessage(ctx, payload, to)
}

// SendTemplate sends a WhatsApp template message.
// parameters map to positional {{1}}, {{2}}, ... variables in the template body.
func (p *Provider) SendTemplate(ctx context.Context, req *provider.SendRequest, templateName string, parameters []string) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}
	if templateName == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "template name is required",
		}, nil
	}
	lang := "en_US"
	if req.Language != "" {
		lang = req.Language
	}

	components := []map[string]any{}
	if len(parameters) > 0 {
		params := make([]map[string]any, 0, len(parameters))
		for _, v := range parameters {
			params = append(params, map[string]any{"type": "text", "text": v})
		}
		components = append(components, map[string]any{
			"type":       "body",
			"parameters": params,
		})
	}

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                stripPhonePrefix(req.RecipientPhone),
		"type":              "template",
		"template": map[string]any{
			"name":       templateName,
			"language":   map[string]any{"code": lang},
			"components": components,
		},
	}
	return p.postMessage(ctx, payload, stripPhonePrefix(req.RecipientPhone))
}

// SendMedia sends a WhatsApp message with a media URL and optional caption.
// mediaType is one of "image", "video", "audio", "document" (passed literally).
func (p *Provider) SendMedia(ctx context.Context, req *provider.SendRequest, mediaURL, mediaType string) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}
	if mediaURL == "" || mediaType == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "mediaURL and mediaType are required",
		}, nil
	}
	obj := map[string]any{"link": mediaURL}
	if req.Body != "" {
		obj["caption"] = req.Body
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                stripPhonePrefix(req.RecipientPhone),
		"type":              mediaType,
		mediaType:           obj,
	}
	return p.postMessage(ctx, payload, stripPhonePrefix(req.RecipientPhone))
}

// GetStatus returns StatusSent — Meta delivers status via Webhook; pull-based
// per-message lookup is not exposed as a simple API.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusSent}, nil
}

func (p *Provider) postMessage(ctx context.Context, payload map[string]any, recipient string) (*provider.SendResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("meta: marshal: %w", err)
	}
	url := fmt.Sprintf("%s/%s/messages", p.cfg.GraphAPIBase, p.cfg.PhoneNumberID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("meta: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.AccessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.logger.Error("meta: http failed", map[string]any{"error": err.Error(), "recipient": MaskPhone(recipient)})
		return &provider.SendResponse{Success: false, ProviderCode: ProviderCode, Error: err.Error()}, nil
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("meta: read body: %w", err)
	}

	var parsed MessagesResponse
	_ = json.Unmarshal(raw, &parsed)

	if resp.StatusCode >= 300 || parsed.Error != nil {
		errMsg := ""
		if parsed.Error != nil {
			errMsg = fmt.Sprintf("%d %s (%s)", parsed.Error.Code, parsed.Error.Message, parsed.Error.Type)
		} else {
			errMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(raw))
		}
		p.logger.Warn("meta: non-success response", map[string]any{
			"status":    resp.StatusCode,
			"recipient": MaskPhone(recipient),
			"error":     errMsg,
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        errMsg,
			RawResponse:  raw,
		}, nil
	}

	var messageID string
	if len(parsed.Messages) > 0 {
		messageID = parsed.Messages[0].ID
	}
	p.logger.Info("meta: sent", map[string]any{
		"recipient":  MaskPhone(recipient),
		"message_id": messageID,
	})
	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: messageID,
		RawResponse:       raw,
	}, nil
}

// MaskPhone masks a phone number for safe logging.
func MaskPhone(phone string) string {
	phone = stripPhonePrefix(phone)
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:len(phone)-4] + "****"
}

func stripPhonePrefix(s string) string {
	s = strings.TrimPrefix(s, "whatsapp:")
	s = strings.TrimPrefix(s, "+")
	return s
}
