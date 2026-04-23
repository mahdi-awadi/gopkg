// Package twilio provides a Twilio Voice implementation of
// communication/provider.Provider — flash-call voice notifications
// (brief ring-then-hangup for OTP-style verification).
package twilio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	twilioGo "github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "twilio_voice"

// DefaultRejectURL is a TwiML endpoint that immediately rejects the call,
// producing the "ring once" flash-call behavior typical of OTP delivery.
const DefaultRejectURL = "https://twimlets.com/reject"

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

// Config holds static Twilio credentials and the caller identity.
type Config struct {
	AccountSID string
	AuthToken  string
	FromNumber string
	// RejectURL is the TwiML URL used for flash-call behavior.
	// Leave empty to use DefaultRejectURL.
	RejectURL string
}

// Provider implements provider.Provider on top of Twilio Voice.
type Provider struct {
	cfg    Config
	client *twilioGo.RestClient
	logger Logger
}

// New constructs a Twilio voice Provider. logger may be nil (becomes noop).
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	client := twilioGo.NewRestClientWithParams(twilioGo.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	})
	if cfg.RejectURL == "" {
		cfg.RejectURL = DefaultRejectURL
	}
	return &Provider{cfg: cfg, client: client, logger: logger}
}

// Compile-time check.
var _ provider.Provider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelVoice}
}

// ValidateConfig is a no-op — empty creds are allowed; per-tenant
// credentials can be layered via a consumer wrapper.
func (p *Provider) ValidateConfig() error { return nil }

// Enabled returns true when all static credentials are populated.
func (p *Provider) Enabled() bool {
	return p.cfg.AccountSID != "" && p.cfg.AuthToken != "" && p.cfg.FromNumber != ""
}

// Send initiates a Twilio voice call to req.RecipientPhone. For flash-call
// (ring-once) delivery leave req.Body empty and the default RejectURL
// TwiML will drop the call as the recipient's phone rings.
func (p *Provider) Send(_ context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}

	params := &twilioApi.CreateCallParams{}
	params.SetTo(req.RecipientPhone)
	params.SetFrom(p.cfg.FromNumber)
	params.SetUrl(p.cfg.RejectURL)

	resp, err := p.client.Api.CreateCall(params)
	if err != nil {
		p.logger.Error("twilio voice: create call failed", map[string]any{
			"recipient": MaskPhone(req.RecipientPhone),
			"error":     err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}

	raw, _ := json.Marshal(resp)
	var callSID string
	if resp.Sid != nil {
		callSID = *resp.Sid
	}

	p.logger.Info("twilio voice: call initiated", map[string]any{
		"recipient": MaskPhone(req.RecipientPhone),
		"call_sid":  callSID,
	})

	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: callSID,
		RawResponse:       raw,
	}, nil
}

// GetStatus fetches the current status of a call by its Twilio SID.
func (p *Provider) GetStatus(_ context.Context, callSID string) (*provider.DeliveryStatus, error) {
	resp, err := p.client.Api.FetchCall(callSID, &twilioApi.FetchCallParams{})
	if err != nil {
		return nil, fmt.Errorf("twilio voice: fetch call %s: %w", callSID, err)
	}

	var status provider.Status
	if resp.Status != nil {
		switch *resp.Status {
		case "queued", "ringing":
			status = provider.StatusQueued
		case "in-progress":
			status = provider.StatusSent
		case "completed":
			status = provider.StatusDelivered
		case "failed", "busy", "no-answer", "canceled":
			status = provider.StatusFailed
		default:
			status = provider.StatusUnknown
		}
	}
	raw, _ := json.Marshal(resp)
	return &provider.DeliveryStatus{
		MessageID:   callSID,
		Status:      status,
		RawResponse: raw,
	}, nil
}

// MaskPhone masks the final 4 digits of a phone number for logging.
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:len(phone)-4] + "****"
}
