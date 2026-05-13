package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mahdi-awadi/gopkg/communication/telegram/bot/client"
)

// Session holds per-user auth and preference data.
type Session struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresAt    int64    `json:"expires_at"`
	Roles        []string `json:"roles"`
	UserID       string   `json:"user_id"`
	UserType     string   `json:"user_type"`
	Lang         string   `json:"lang"`
}

// IsExpired returns true if the token is expired or about to expire (5 min buffer).
func (s *Session) IsExpired() bool {
	return time.Now().Unix() >= s.ExpiresAt-300
}

// IsAdmin returns true if the user has admin-level roles.
func (s *Session) IsAdmin() bool {
	return s.UserType == "admin" || s.UserType == "superadmin"
}

// SessionManager handles session CRUD in Redis.
type SessionManager struct {
	redis    *redis.Client
	identity *client.IdentityClient
}

func NewSessionManager(rdb *redis.Client, identity *client.IdentityClient) *SessionManager {
	return &SessionManager{redis: rdb, identity: identity}
}

func sessionKey(partnerID string, telegramID int64) string {
	return fmt.Sprintf("tgbot:session:%s:%d", partnerID, telegramID)
}

// GetSession loads a session from Redis.
func (m *SessionManager) GetSession(ctx context.Context, partnerID string, telegramID int64) (*Session, error) {
	data, err := m.redis.Get(ctx, sessionKey(partnerID, telegramID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return &s, nil
}

// GetValidToken returns a non-expired token, re-exchanging if needed.
func (m *SessionManager) GetValidToken(ctx context.Context, partnerID string, telegramID int64) (string, *Session, error) {
	sess, err := m.GetSession(ctx, partnerID, telegramID)
	if err != nil {
		return "", nil, err
	}
	if sess == nil {
		return "", nil, nil
	}
	if !sess.IsExpired() {
		return sess.Token, sess, nil
	}
	// Token expired, re-exchange
	newSess, err := m.Authenticate(ctx, partnerID, telegramID)
	if err != nil {
		return "", nil, fmt.Errorf("re-authenticate: %w", err)
	}
	return newSess.Token, newSess, nil
}

// Authenticate exchanges the Telegram ID for JWT and stores the session.
func (m *SessionManager) Authenticate(ctx context.Context, partnerID string, telegramID int64) (*Session, error) {
	resp, err := m.identity.ExchangeTelegramToken(
		fmt.Sprintf("%d", telegramID),
		partnerID,
	)
	if err != nil {
		return nil, fmt.Errorf("exchange token: %w", err)
	}

	sess := &Session{
		Token:        resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second).Unix(),
		Roles:        resp.Roles,
		UserID:       resp.User.ID,
		UserType:     resp.User.UserType,
		Lang:         "ar",
	}

	if err := m.saveSession(ctx, partnerID, telegramID, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// HandleUnauthorized re-authenticates and retries the callback.
func (m *SessionManager) HandleUnauthorized(ctx context.Context, partnerID string, telegramID int64, fn func(token string) error) error {
	sess, err := m.Authenticate(ctx, partnerID, telegramID)
	if err != nil {
		return err
	}
	return fn(sess.Token)
}

// SetLang updates the language preference in the session.
func (m *SessionManager) SetLang(ctx context.Context, partnerID string, telegramID int64, lang string) error {
	sess, err := m.GetSession(ctx, partnerID, telegramID)
	if err != nil || sess == nil {
		return fmt.Errorf("session not found")
	}
	sess.Lang = lang
	return m.saveSession(ctx, partnerID, telegramID, sess)
}

func (m *SessionManager) saveSession(ctx context.Context, partnerID string, telegramID int64, sess *Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	ttl := time.Until(time.Unix(sess.ExpiresAt, 0))
	if ttl < time.Minute {
		ttl = time.Minute
	}
	return m.redis.Set(ctx, sessionKey(partnerID, telegramID), data, ttl).Err()
}
