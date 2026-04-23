// Package telegram provides a Telegram Bot implementation of
// communication/provider.Provider. Uses Telegram's Bot API directly
// over HTTPS — no third-party Telegram client required.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

// ProviderCode is the code used in the Registry / logs.
const ProviderCode = "telegram_bot"

// APIBaseURL is the default Telegram Bot API base.
const APIBaseURL = "https://api.telegram.org"

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

// Config holds the bot token.
type Config struct {
	BotToken string
	// APIBaseURL overrides the default (useful for test servers). Leave empty for the default.
	APIBaseURL string
	// Timeout for individual HTTP requests. Zero means 10 seconds.
	Timeout time.Duration
}

// APIResponse is the envelope all Telegram Bot API calls return.
type APIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	Description string          `json:"description,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
}

// Provider implements provider.Provider via Telegram Bot API.
type Provider struct {
	cfg    Config
	client *http.Client
	logger Logger
}

// New constructs a Telegram Provider. logger may be nil (becomes noop).
func New(cfg Config, logger Logger) *Provider {
	if logger == nil {
		logger = noopLogger{}
	}
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = APIBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Provider{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		logger: logger,
	}
}

// Compile-time check.
var _ provider.Provider = (*Provider)(nil)

// Code returns the provider identifier.
func (p *Provider) Code() string { return ProviderCode }

// SupportedChannels returns the channels this provider supports.
func (p *Provider) SupportedChannels() []provider.Channel {
	return []provider.Channel{provider.ChannelTelegram}
}

// ValidateConfig returns non-nil if BotToken is empty.
func (p *Provider) ValidateConfig() error {
	if p.cfg.BotToken == "" {
		return fmt.Errorf("telegram: BotToken is required")
	}
	return nil
}

// Enabled returns true when BotToken is set.
func (p *Provider) Enabled() bool { return p.cfg.BotToken != "" }

// Send sends a text message to req.RecipientTelegramChatID.
func (p *Provider) Send(ctx context.Context, req *provider.SendRequest) (*provider.SendResponse, error) {
	if req.RecipientTelegramChatID == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "recipient chat_id is required",
		}, nil
	}
	if p.cfg.BotToken == "" {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        "bot token not configured",
		}, nil
	}

	// Prefer HTML if provided, else plain text.
	text := req.Body
	parseMode := ""
	if req.HTMLBody != "" {
		text = req.HTMLBody
		parseMode = "HTML"
	}

	form := url.Values{}
	form.Set("chat_id", req.RecipientTelegramChatID)
	form.Set("text", text)
	if parseMode != "" {
		form.Set("parse_mode", parseMode)
	}

	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", p.cfg.APIBaseURL, p.cfg.BotToken)
	resp, err := p.call(ctx, endpoint, form)
	if err != nil {
		p.logger.Error("telegram: send failed", map[string]any{
			"chat_id": req.RecipientTelegramChatID,
			"error":   err.Error(),
		})
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        err.Error(),
		}, nil
	}
	if !resp.OK {
		return &provider.SendResponse{
			Success:      false,
			ProviderCode: ProviderCode,
			Error:        fmt.Sprintf("telegram: %d %s", resp.ErrorCode, resp.Description),
		}, nil
	}

	p.logger.Info("telegram: sent", map[string]any{
		"chat_id": req.RecipientTelegramChatID,
	})

	return &provider.SendResponse{
		Success:      true,
		ProviderCode: ProviderCode,
		RawResponse:  resp.Result,
	}, nil
}

// GetStatus isn't meaningful for Telegram send operations; returns StatusSent.
func (p *Provider) GetStatus(_ context.Context, messageID string) (*provider.DeliveryStatus, error) {
	return &provider.DeliveryStatus{MessageID: messageID, Status: provider.StatusSent}, nil
}

func (p *Provider) call(ctx context.Context, endpoint string, form url.Values) (*APIResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.URL.RawQuery = form.Encode()

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var api APIResponse
	if err := json.Unmarshal(body, &api); err != nil {
		return nil, fmt.Errorf("decode: %w (body: %s)", err, string(body))
	}
	return &api, nil
}
