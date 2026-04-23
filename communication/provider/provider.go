// Package provider defines a cross-channel notification-delivery contract.
//
// The Provider interface captures the shape every delivery adapter
// (SMS, email, push, WhatsApp, voice, etc.) must satisfy. Channel-specific
// extension interfaces (EmailProvider, SMSProvider, PushProvider,
// WhatsAppProvider) add features that only apply to that channel.
//
// This package is stdlib-only. Concrete adapters ship in sibling packages.
package provider

import "context"

// Channel identifies a delivery channel.
type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelPush     Channel = "push"
	ChannelWhatsApp Channel = "whatsapp"
	ChannelTelegram Channel = "telegram"
	ChannelVoice    Channel = "voice"
)

// Status is the lifecycle state of a sent message as reported by a provider.
type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusQueued    Status = "queued"
	StatusSent      Status = "sent"
	StatusDelivered Status = "delivered"
	StatusRead      Status = "read"
	StatusFailed    Status = "failed"
)

// SendRequest is the envelope passed to Provider.Send.
//
// Recipient fields are channel-specific; only the relevant ones need to be
// populated for a given provider (email → RecipientEmail, SMS/voice →
// RecipientPhone, push → RecipientDeviceToken, Telegram → RecipientTelegramChatID).
type SendRequest struct {
	// Recipient identifiers (populate whichever fits the channel).
	RecipientEmail          string
	RecipientPhone          string
	RecipientDeviceToken    string
	RecipientTelegramChatID string

	// Content.
	Subject  string
	Body     string
	HTMLBody string

	// Metadata.
	TemplateCode  string
	Language      string
	CorrelationID string
	ContextData   map[string]string

	// Options is a provider-specific pass-through for settings the canonical
	// envelope does not cover. Keys and values are adapter-defined.
	Options map[string]any
}

// SendResponse is the result of a Send operation.
type SendResponse struct {
	Success           bool
	ProviderCode      string
	ProviderMessageID string
	Error             string
	RawResponse       []byte
}

// DeliveryStatus is the result of Provider.GetStatus.
type DeliveryStatus struct {
	MessageID    string
	Status       Status
	DeliveredAt  *string
	FailedAt     *string
	ErrorCode    string
	ErrorMessage string
	RawResponse  []byte
}

// Provider is the base contract every delivery adapter implements.
//
// Implementations MUST be safe for concurrent use. Every I/O method
// honors ctx cancellation.
type Provider interface {
	// Code returns the unique code identifying this provider (e.g. "sendgrid").
	Code() string

	// SupportedChannels returns the channels this provider supports.
	SupportedChannels() []Channel

	// Send delivers the request through the provider.
	Send(ctx context.Context, req *SendRequest) (*SendResponse, error)

	// GetStatus retrieves the current delivery status for a message.
	GetStatus(ctx context.Context, messageID string) (*DeliveryStatus, error)

	// ValidateConfig returns non-nil if provider configuration is invalid.
	ValidateConfig() error

	// Enabled returns whether this provider is properly configured and ready.
	Enabled() bool
}

// Attachment is an email attachment payload.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// EmailProvider adds email-specific methods on top of Provider.
type EmailProvider interface {
	Provider
	SendWithAttachments(ctx context.Context, req *SendRequest, attachments []Attachment) (*SendResponse, error)
}

// SMSProvider adds SMS-specific methods on top of Provider.
type SMSProvider interface {
	Provider
	SupportedCountries() []string
	CostEstimate(ctx context.Context, phoneNumber string) (amount float64, currency string, err error)
}

// PushProvider adds push-specific methods on top of Provider.
type PushProvider interface {
	Provider
	SendToTopic(ctx context.Context, topic string, req *SendRequest) (*SendResponse, error)
	SendMulticast(ctx context.Context, tokens []string, req *SendRequest) ([]*SendResponse, error)
}

// WhatsAppProvider adds WhatsApp-specific methods on top of Provider.
type WhatsAppProvider interface {
	Provider
	SendTemplate(ctx context.Context, req *SendRequest, templateName string, parameters []string) (*SendResponse, error)
	SendMedia(ctx context.Context, req *SendRequest, mediaURL, mediaType string) (*SendResponse, error)
}

// ProviderError is a structured error from a delivery provider.
type ProviderError struct {
	ProviderCode string
	Code         string
	Message      string
	Retryable    bool
	RawError     error
}

// Error implements error.
func (e *ProviderError) Error() string {
	if e.RawError != nil {
		return e.ProviderCode + ": " + e.Message + " (" + e.RawError.Error() + ")"
	}
	return e.ProviderCode + ": " + e.Message
}

// Unwrap supports errors.Is / errors.As.
func (e *ProviderError) Unwrap() error { return e.RawError }

// NewProviderError constructs a ProviderError.
func NewProviderError(code, errCode, message string, retryable bool, raw error) *ProviderError {
	return &ProviderError{
		ProviderCode: code,
		Code:         errCode,
		Message:      message,
		Retryable:    retryable,
		RawError:     raw,
	}
}
