package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const stateTTL = 30 * time.Minute

// Manager handles conversation state in Redis.
type Manager struct {
	redis *redis.Client
}

func NewManager(rdb *redis.Client) *Manager {
	return &Manager{redis: rdb}
}

func stateKey(partnerID string, telegramID int64) string {
	return fmt.Sprintf("tgbot:state:%s:%d", partnerID, telegramID)
}

// Get returns the current conversation state, or nil if none.
func (m *Manager) Get(ctx context.Context, partnerID string, telegramID int64) (*ConversationState, error) {
	data, err := m.redis.Get(ctx, stateKey(partnerID, telegramID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}
	var s ConversationState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	return &s, nil
}

// Set saves the conversation state.
func (m *Manager) Set(ctx context.Context, partnerID string, telegramID int64, s *ConversationState) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return m.redis.Set(ctx, stateKey(partnerID, telegramID), data, stateTTL).Err()
}

// Clear deletes the conversation state.
func (m *Manager) Clear(ctx context.Context, partnerID string, telegramID int64) error {
	return m.redis.Del(ctx, stateKey(partnerID, telegramID)).Err()
}
