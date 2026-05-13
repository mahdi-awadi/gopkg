package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IdentityClient calls identity-service directly (internal, no gateway).
type IdentityClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

func NewIdentityClient(baseURL, serviceKey string) *IdentityClient {
	return &IdentityClient{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// AuthResponse is the response from ExchangeTelegramToken.
type AuthResponse struct {
	Success      bool      `json:"success"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int32     `json:"expiresIn"`
	Roles        []string  `json:"roles"`
	User         UserInfo  `json:"user"`
	Error        *APIError `json:"error,omitempty"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserType string `json:"userType"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PartnerBotConfig is a partner's bot token info.
type PartnerBotConfig struct {
	PartnerID   string `json:"partnerId"`
	PartnerCode string `json:"partnerCode"`
	BotToken    string `json:"botToken"`
}

type getBotsResponse struct {
	Success bool               `json:"success"`
	Bots    []PartnerBotConfig `json:"bots"`
	Error   *APIError          `json:"error,omitempty"`
}

// ExchangeTelegramToken exchanges a Telegram user ID for JWT tokens.
func (c *IdentityClient) ExchangeTelegramToken(telegramID, partnerID string) (*AuthResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"telegramId": telegramID,
		"partnerId":  partnerID,
	})

	resp, err := c.doInternal(
		"/eticket.identity.v1.IdentityService/ExchangeTelegramToken",
		body,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ExchangeTelegramToken response: %w", err)
	}
	if !result.Success && result.Error != nil {
		return nil, fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
	}
	return &result, nil
}

// GetTelegramBots fetches all active partner bot tokens.
func (c *IdentityClient) GetTelegramBots() ([]PartnerBotConfig, error) {
	resp, err := c.doInternal(
		"/eticket.identity.v1.IdentityService/GetTelegramBots",
		[]byte("{}"),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result getBotsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode GetTelegramBots response: %w", err)
	}
	if !result.Success && result.Error != nil {
		return nil, fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
	}
	return result.Bots, nil
}

func (c *IdentityClient) doInternal(path string, body []byte) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", c.serviceKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("request %s returned %d: %s", path, resp.StatusCode, string(respBody))
	}
	return resp, nil
}
