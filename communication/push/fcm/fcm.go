// Package fcm provides a Firebase Cloud Messaging implementation of
// communication/provider.PushProvider.
package fcm

import (
	"context"
	"encoding/json"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/mahdi-awadi/gopkg/communication/provider"
	"google.golang.org/api/option"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "firebase_fcm"

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

// Config holds Firebase project settings.
type Config struct {
	ProjectID       string
	CredentialsJSON string
}

// Provider implements provider.PushProvider via Firebase Cloud Messaging.
type Provider struct {
	cfg    Config
	client *messaging.Client
	logger Logger
}

// New constructs an FCM Provider. logger may be nil (becomes noop).
func New(ctx context.Context, cfg Config, logger Logger) (*Provider, error) {
	if logger == nil {
		logger = noopLogger{}
	}
	if cfg.ProjectID == "" || cfg.CredentialsJSON == "" {
		return nil, fmt.Errorf("fcm: ProjectID and CredentialsJSON are required")
	}

	opt := option.WithCredentialsJSON([]byte(cfg.CredentialsJSON))
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, opt)
	if err != nil {
		return nil, fmt.Errorf("fcm: initialize app: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: messaging client: %w", err)
	}
	return &Provider{cfg: cfg, client: client, logger: logger}, nil
}

// Compile-time check.
var _ provider.PushProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelPush}
}

// ValidateConfig returns non-nil if the provider is not ready.
func (p *Provider) ValidateConfig() error {
	if p.cfg.ProjectID == "" {
		return fmt.Errorf("fcm: ProjectID required")
	}
	if p.cfg.CredentialsJSON == "" {
		return fmt.Errorf("fcm: CredentialsJSON required")
	}
	return nil
}

// Enabled returns true when the client was successfully initialized.
func (p *Provider) Enabled() bool { return p.client != nil }

// Send sends a push notification to a single device token.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientDeviceToken == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "device token is required",
		}, nil
	}

	m := &messaging.Message{
		Token: req.RecipientDeviceToken,
		Notification: &messaging.Notification{
			Title: req.Subject,
			Body:  req.Body,
		},
		Data:    req.ContextData,
		Android: defaultAndroid(),
		APNS:    defaultAPNS(),
	}

	resp, err := p.client.Send(ctx, m)
	if err != nil {
		p.logger.Error("fcm: send failed", map[string]any{
			"token": MaskToken(req.RecipientDeviceToken),
			"error": err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}

	raw, _ := json.Marshal(map[string]string{"message_id": resp})
	p.logger.Info("fcm: sent", map[string]any{"message_id": resp, "title": req.Subject})
	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: resp,
		RawResponse:       raw,
	}, nil
}

// SendToTopic fans out a push notification to a topic.
func (p *Provider) SendToTopic(ctx context.Context, topic string, req *provider.SendRequest) (*provider.SendResponse, error) {
	m := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: req.Subject,
			Body:  req.Body,
		},
		Data: req.ContextData,
	}
	resp, err := p.client.Send(ctx, m)
	if err != nil {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}
	raw, _ := json.Marshal(map[string]string{"message_id": resp, "topic": topic})
	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: resp,
		RawResponse:       raw,
	}, nil
}

// SendMulticast sends a push notification to multiple device tokens.
func (p *Provider) SendMulticast(ctx context.Context, tokens []string, req *provider.SendRequest) ([]*provider.SendResponse, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("fcm: at least one token is required")
	}
	m := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: req.Subject,
			Body:  req.Body,
		},
		Data:    req.ContextData,
		Android: defaultAndroid(),
		APNS:    defaultAPNS(),
	}
	batch, err := p.client.SendEachForMulticast(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("fcm: multicast: %w", err)
	}
	out := make([]*provider.SendResponse, len(batch.Responses))
	for i, r := range batch.Responses {
		if r.Success {
			out[i] = &provider.SendResponse{
				Success:           true,
				ProviderCode:      ProviderCode,
				ProviderMessageID: r.MessageID,
			}
		} else {
			out[i] = &provider.SendResponse{
				Success:      false,
				ProviderCode: ProviderCode,
				Error:        r.Error.Error(),
			}
		}
	}
	p.logger.Info("fcm: multicast complete", map[string]any{
		"total":   len(tokens),
		"success": batch.SuccessCount,
		"failure": batch.FailureCount,
	})
	return out, nil
}

// GetStatus returns StatusSent; FCM has no per-message lookup API.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusSent}, nil
}

// MaskToken masks a device token for logging, keeping the first 8 chars.
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:8] + "****"
}

func defaultAndroid() *messaging.AndroidConfig {
	return &messaging.AndroidConfig{
		Priority: "high",
		Notification: &messaging.AndroidNotification{
			Sound: "default",
		},
	}
}

func defaultAPNS() *messaging.APNSConfig {
	badge := 1
	return &messaging.APNSConfig{
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Sound: "default",
				Badge: &badge,
			},
		},
	}
}
