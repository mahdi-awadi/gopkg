// Package twilio provides a WhatsApp-over-Twilio implementation of
// communication/provider.WhatsAppProvider. Uses Twilio's Messaging API
// with the "whatsapp:" channel prefix.
package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	twilioGo "github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "twilio_whatsapp"

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

// Config holds Twilio credentials and the WhatsApp From identity.
type Config struct {
	AccountSID string
	AuthToken  string
	// From is the Twilio WhatsApp sender (e.g. "+15005550006" —
	// the adapter prefixes "whatsapp:" automatically).
	From string
	// ContentSid is the optional Content Template SID used by SendTemplate.
	ContentSid string
}

// Provider implements provider.WhatsAppProvider via Twilio Messaging.
type Provider struct {
	cfg    Config
	client *twilioGo.RestClient
	logger Logger
}

// New constructs a WhatsApp-Twilio Provider. logger may be nil.
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	client := twilioGo.NewRestClientWithParams(twilioGo.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	})
	return &Provider{cfg: cfg, client: client, logger: logger}
}

// Compile-time check.
var _ provider.WhatsAppProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelWhatsApp}
}

// ValidateConfig is a no-op; callers validate Enabled before Send.
func (p *Provider) ValidateConfig() error { return nil }

// Enabled returns true when creds and From are present.
func (p *Provider) Enabled() bool {
	return p.cfg.AccountSID != "" && p.cfg.AuthToken != "" && p.cfg.From != ""
}

// Send sends a plain WhatsApp text message.
func (p *Provider) Send(_ context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}
	params := &twilioApi.CreateMessageParams{}
	params.SetFrom(withWhatsAppPrefix(p.cfg.From))
	params.SetTo(withWhatsAppPrefix(req.RecipientPhone))
	params.SetBody(req.Body)

	resp, err := p.client.Api.CreateMessage(params)
	return p.buildResponse(req.RecipientPhone, resp, err)
}

// SendTemplate sends a template message using Twilio Content API.
// templateName is ignored (Twilio uses a Content SID); instead, set
// Config.ContentSid or pass it in via the `content_sid` option.
func (p *Provider) SendTemplate(_ context.Context, req *provider.SendRequest, _ string, parameters []string) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}
	contentSid := p.cfg.ContentSid
	if req.Options != nil {
		if v, ok := req.Options["content_sid"].(string); ok && v != "" {
			contentSid = v
		}
	}
	if contentSid == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "content_sid not configured",
		}, nil
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetFrom(withWhatsAppPrefix(p.cfg.From))
	params.SetTo(withWhatsAppPrefix(req.RecipientPhone))
	params.SetContentSid(contentSid)
	if len(parameters) > 0 {
		vars := map[string]string{}
		for i, v := range parameters {
			vars[fmt.Sprintf("%d", i+1)] = v
		}
		if b, err := json.Marshal(vars); err == nil {
			params.SetContentVariables(string(b))
		}
	}

	resp, err := p.client.Api.CreateMessage(params)
	return p.buildResponse(req.RecipientPhone, resp, err)
}

// SendMedia sends a WhatsApp message with a media URL attached.
func (p *Provider) SendMedia(_ context.Context, req *provider.SendRequest, mediaURL, _ string) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}
	params := &twilioApi.CreateMessageParams{}
	params.SetFrom(withWhatsAppPrefix(p.cfg.From))
	params.SetTo(withWhatsAppPrefix(req.RecipientPhone))
	if req.Body != "" {
		params.SetBody(req.Body)
	}
	if mediaURL != "" {
		params.SetMediaUrl([]string{mediaURL})
	}

	resp, err := p.client.Api.CreateMessage(params)
	return p.buildResponse(req.RecipientPhone, resp, err)
}

// GetStatus fetches delivery status via Twilio's FetchMessage.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	resp, err := p.client.Api.FetchMessage(messageID, &twilioApi.FetchMessageParams{})
	if err != nil {
		return nil, fmt.Errorf("whatsapp-twilio: fetch %s: %w", messageID, err)
	}
	var status provider.Status
	if resp.Status != nil {
		switch *resp.Status {
		case "queued", "sending":
			status = provider.StatusQueued
		case "sent":
			status = provider.StatusSent
		case "delivered":
			status = provider.StatusDelivered
		case "read":
			status = provider.StatusRead
		case "undelivered", "failed":
			status = provider.StatusFailed
		default:
			status = provider.StatusUnknown
		}
	}
	raw, _ := json.Marshal(resp)
	return &provider.DeliveryStatus{MessageID: messageID, Status: status, RawResponse: raw}, nil
}

// MaskPhone masks a phone number for safe logging.
func MaskPhone(phone string) string {
	phone = strings.TrimPrefix(phone, "whatsapp:")
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:len(phone)-4] + "****"
}

func withWhatsAppPrefix(addr string) string {
	if strings.HasPrefix(addr, "whatsapp:") {
		return addr
	}
	return "whatsapp:" + addr
}

func (p *Provider) buildResponse(recipient string, resp *twilioApi.ApiV2010Message, err error) (*provider.SendResponse, error) {
	if err != nil {
		p.logger.Error("whatsapp-twilio: send failed", map[string]any{
			"recipient": MaskPhone(recipient),
			"error":     err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}
	raw, _ := json.Marshal(resp)
	var sid string
	if resp.Sid != nil {
		sid = *resp.Sid
	}
	p.logger.Info("whatsapp-twilio: sent", map[string]any{
		"recipient":  MaskPhone(recipient),
		"message_id": sid,
	})
	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: sid,
		RawResponse:       raw,
	}, nil
}
