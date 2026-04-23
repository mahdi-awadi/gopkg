// Package expo provides an Expo Push Notifications implementation of
// communication/provider.PushProvider.
//
// Uses the public Expo Push API (exp.host) — no credentials required.
package expo

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
const ProviderCode = "expo_push"

// ExpoPushAPIURL is the default Expo push endpoint.
const ExpoPushAPIURL = "https://exp.host/--/api/v2/push/send"

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

// PushMessage is the canonical Expo push payload.
type PushMessage struct {
	To        string         `json:"to"`
	Title     string         `json:"title,omitempty"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data,omitempty"`
	Sound     string         `json:"sound,omitempty"`
	Badge     *int           `json:"badge,omitempty"`
	ChannelID string         `json:"channelId,omitempty"`
	Priority  string         `json:"priority,omitempty"`
	TTL       int            `json:"ttl,omitempty"`
}

// PushTicket is a single ticket returned by the Expo push API.
type PushTicket struct {
	ID      string         `json:"id,omitempty"`
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// PushResponse is the raw response envelope.
type PushResponse struct {
	Data []PushTicket `json:"data"`
}

// Provider implements provider.PushProvider via the Expo public push API.
type Provider struct {
	endpoint   string
	httpClient *http.Client
	logger     Logger
}

// New constructs an Expo Provider. logger may be nil (becomes noop).
func New(logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	return &Provider{
		endpoint:   ExpoPushAPIURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// Compile-time check.
var _ provider.PushProvider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelPush}
}

// ValidateConfig is a no-op (Expo push requires no credentials).
func (p *Provider) ValidateConfig() error { return nil }

// Enabled always returns true (no creds required).
func (p *Provider) Enabled() bool { return true }

// IsExpoToken reports whether s looks like an Expo push token.
func IsExpoToken(s string) bool {
	return strings.HasPrefix(s, "ExponentPushToken[") || strings.HasPrefix(s, "ExpoPushToken[")
}

// Send sends a push to a single Expo token.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientDeviceToken == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "device token is required",
		}, nil
	}
	if !IsExpoToken(req.RecipientDeviceToken) {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "not an Expo push token",
		}, nil
	}

	m := buildMessage(req.RecipientDeviceToken, req)
	tickets, err := p.sendBatch(ctx, []PushMessage{m})
	if err != nil {
		p.logger.Error("expo: send failed", map[string]any{
			"token": MaskToken(req.RecipientDeviceToken),
			"error": err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}
	if len(tickets) == 0 {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "no ticket returned from Expo",
		}, nil
	}

	t := tickets[0]
	if t.Status == "error" {
		p.logger.Warn("expo: ticket error", map[string]any{"message": t.Message})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        t.Message,
		}, nil
	}

	raw, _ := json.Marshal(t)
	p.logger.Info("expo: sent", map[string]any{"ticket_id": t.ID, "title": req.Subject})
	return &provider.SendResponse{
		Success:           true,
		ProviderCode:      ProviderCode,
		ProviderMessageID: t.ID,
		RawResponse:       raw,
	}, nil
}

// SendMulticast sends a push to multiple Expo tokens. Non-Expo tokens are skipped.
func (p *Provider) SendMulticast(ctx context.Context, tokens []string, req *provider.SendRequest) ([]*provider.SendResponse, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("expo: at least one token is required")
	}
	messages := make([]PushMessage, 0, len(tokens))
	for _, t := range tokens {
		if !IsExpoToken(t) {
			continue
		}
		messages = append(messages, buildMessage(t, req))
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("expo: no valid Expo tokens in batch")
	}

	tickets, err := p.sendBatch(ctx, messages)
	if err != nil {
		return nil, err
	}

	out := make([]*provider.SendResponse, len(tickets))
	for i, t := range tickets {
		if t.Status == "error" {
			out[i] = &provider.SendResponse{
				Success:      false,
				ProviderCode: ProviderCode,
				Error:        t.Message,
			}
		} else {
			out[i] = &provider.SendResponse{
				Success:           true,
				ProviderCode:      ProviderCode,
				ProviderMessageID: t.ID,
			}
		}
	}
	return out, nil
}

// SendToTopic is not supported by the Expo public push API.
func (p *Provider) SendToTopic(_ context.Context, _ string, _ *provider.SendRequest) (*provider.SendResponse, error) {
	return nil, fmt.Errorf("expo: topic-based sending not supported")
}

// GetStatus returns StatusSent (Expo per-ticket status lookup requires the Receipts API which is not implemented here).
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusSent}, nil
}

// MaskToken masks an Expo token for logging.
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:8] + "****"
}

func buildMessage(token string, req *provider.SendRequest) PushMessage {
	m := PushMessage{
		To:       token,
		Title:    req.Subject,
		Body:     req.Body,
		Sound:    "default",
		Priority: "high",
	}
	if len(req.ContextData) > 0 {
		data := make(map[string]any, len(req.ContextData))
		for k, v := range req.ContextData {
			data[k] = v
		}
		m.Data = data
	}
	return m
}

func (p *Provider) sendBatch(ctx context.Context, messages []PushMessage) ([]PushTicket, error) {
	body, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("expo: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("expo: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("expo: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("expo: read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("expo: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var decoded PushResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("expo: decode response: %w", err)
	}
	return decoded.Data, nil
}
