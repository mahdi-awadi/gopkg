// Package twilio provides a Twilio SMS implementation of
// communication/provider.Provider (and SMSProvider).
//
// Construct via New; the returned *Provider is safe for concurrent use.
package twilio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "twilio_sms"

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

// Config holds the static Twilio credentials. If any of these are empty,
// Enabled() returns false.
type Config struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

// Provider implements provider.SMSProvider on top of Twilio.
type Provider struct {
	cfg    Config
	client *twilio.RestClient
	logger Logger

	// Countries is the list of supported country dial codes returned by
	// SupportedCountries. If nil, DefaultCountries is returned.
	Countries []string

	// CountryPricing is the static pricing map consulted by CostEstimate.
	// Keys are country dial codes (e.g. "+1"); values are USD per segment.
	// If nil, DefaultCountryPricing is used.
	CountryPricing map[string]float64
}

// DefaultCountries returns a baseline global list of supported dial codes.
func DefaultCountries() []string {
	return []string{"+1", "+44", "+49", "+33", "+964", "+971", "+966", "+962", "+20", "+91"}
}

// DefaultCountryPricing returns a baseline USD-per-segment pricing map.
// Values are rough; consult Twilio's own pricing API for production billing.
func DefaultCountryPricing() map[string]float64 {
	return map[string]float64{
		"+1":   0.0075,
		"+44":  0.04,
		"+49":  0.065,
		"+964": 0.0758,
		"+971": 0.0322,
		"+966": 0.0424,
	}
}

// New constructs a Twilio SMS Provider. logger may be nil (becomes noop).
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	})
	return &Provider{cfg: cfg, client: client, logger: logger}
}

// Compile-time check that *Provider satisfies provider.SMSProvider.
var _ provider.SMSProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the list of channels supported.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelSMS}
}

// ValidateConfig is a no-op (empty static creds are allowed; per-tenant
// credentials may be resolved at send time by the caller).
func (p *Provider) ValidateConfig() error { return nil }

// Enabled returns true when static credentials are present.
func (p *Provider) Enabled() bool {
	return p.cfg.AccountSID != "" && p.cfg.AuthToken != "" && p.cfg.FromNumber != ""
}

// Send delivers the SMS via Twilio.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientPhone == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient phone is required",
		}, nil
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(req.RecipientPhone)
	params.SetFrom(p.cfg.FromNumber)
	params.SetBody(req.Body)

	resp, err := p.client.Api.CreateMessage(params)
	if err != nil {
		p.logger.Error("twilio sms: send failed", map[string]any{
			"recipient": MaskPhone(req.RecipientPhone),
			"error":     err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}

	rawResponse, _ := json.Marshal(resp)

	var messageID string
	if resp.Sid != nil {
		messageID = *resp.Sid
	}
	var status string
	if resp.Status != nil {
		status = *resp.Status
	}

	success := status == "queued" || status == "sending" || status == "sent" || status == "delivered"

	if !success {
		var errorCode, errorMessage string
		if resp.ErrorCode != nil {
			errorCode = fmt.Sprintf("%d", *resp.ErrorCode)
		}
		if resp.ErrorMessage != nil {
			errorMessage = *resp.ErrorMessage
		}

		p.logger.Warn("twilio sms: non-success status", map[string]any{
			"status":     status,
			"error_code": errorCode,
			"error_msg":  errorMessage,
		})

		return &provider.SendResponse{
			Success:           false,
			ProviderCode:      ProviderCode,
			ProviderMessageID: messageID,
			Error:             fmt.Sprintf("twilio: status %s: %s", status, errorMessage),
			RawResponse:       rawResponse,
		}, nil
	}

	p.logger.Info("twilio sms: sent", map[string]any{
		"recipient":  MaskPhone(req.RecipientPhone),
		"message_id": messageID,
		"status":     status,
	})

	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: messageID,
		RawResponse:       rawResponse,
	}, nil
}

// GetStatus looks up delivery status for a message by Twilio SID.
func (p *Provider) GetStatus(ctx context.Context, messageID string) (*provider.DeliveryStatus, error) {
	params := &twilioApi.FetchMessageParams{}

	resp, err := p.client.Api.FetchMessage(messageID, params)
	if err != nil {
		return nil, fmt.Errorf("twilio sms: fetch %s: %w", messageID, err)
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
		case "undelivered", "failed":
			status = provider.StatusFailed
		default:
			status = provider.StatusUnknown
		}
	}

	result := &provider.DeliveryStatus{
		MessageID: messageID,
		Status:    status,
	}

	if resp.DateSent != nil {
		result.DeliveredAt = resp.DateSent
	}
	if resp.ErrorCode != nil {
		result.ErrorCode = fmt.Sprintf("%d", *resp.ErrorCode)
	}
	if resp.ErrorMessage != nil {
		result.ErrorMessage = *resp.ErrorMessage
	}

	rawResponse, _ := json.Marshal(resp)
	result.RawResponse = rawResponse
	return result, nil
}

// SupportedCountries returns the configured country list or the defaults.
func (p *Provider) SupportedCountries() []string {
	if p.Countries != nil {
		return p.Countries
	}
	return DefaultCountries()
}

// CostEstimate returns a rough USD-per-segment cost for the destination.
// Consult Twilio's Pricing API for accurate production values.
func (p *Provider) CostEstimate(_ context.Context, phoneNumber string) (float64, string, error) {
	pricing := p.CountryPricing
	if pricing == nil {
		pricing = DefaultCountryPricing()
	}
	for code, price := range pricing {
		if len(phoneNumber) > len(code) && phoneNumber[:len(code)] == code {
			return price, "USD", nil
		}
	}
	return 0.05, "USD", nil
}

// MaskPhone masks the final 4 digits of a phone number for logging.
// Exported for parity across adapter packages.
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:len(phone)-4] + "****"
}
