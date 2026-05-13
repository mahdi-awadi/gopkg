package support

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// AgentRegistry maps support agents (partner_id + user_id) to Telegram chat IDs.
// Keys: tgbot:agent:{partner_id}:{user_id} → telegram_chat_id
type AgentRegistry struct {
	redis *redis.Client
}

func NewAgentRegistry(rdb *redis.Client) *AgentRegistry {
	return &AgentRegistry{redis: rdb}
}

func agentKey(partnerID, userID string) string {
	return fmt.Sprintf("tgbot:agent:%s:%s", partnerID, userID)
}

// RegisterAgent stores the mapping from agent user_id to Telegram chat ID.
func (r *AgentRegistry) RegisterAgent(ctx context.Context, partnerID, userID string, telegramChatID int64) error {
	return r.redis.Set(ctx, agentKey(partnerID, userID), telegramChatID, 0).Err()
}

// UnregisterAgent removes the agent mapping.
func (r *AgentRegistry) UnregisterAgent(ctx context.Context, partnerID, userID string) error {
	return r.redis.Del(ctx, agentKey(partnerID, userID)).Err()
}

// GetAgentChatID returns the Telegram chat ID for a specific agent.
func (r *AgentRegistry) GetAgentChatID(ctx context.Context, partnerID, userID string) (int64, error) {
	val, err := r.redis.Get(ctx, agentKey(partnerID, userID)).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// GetAllAgentChatIDs returns all registered agent chat IDs for a partner.
func (r *AgentRegistry) GetAllAgentChatIDs(ctx context.Context, partnerID string) ([]int64, error) {
	pattern := fmt.Sprintf("tgbot:agent:%s:*", partnerID)
	var chatIDs []int64

	iter := r.redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		val, err := r.redis.Get(ctx, iter.Val()).Result()
		if err != nil {
			continue
		}
		chatID, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		chatIDs = append(chatIDs, chatID)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return chatIDs, nil
}

// StoreTicketMessage maps a Telegram message ID to a ticket ID (for reply-to-message flow).
func (r *AgentRegistry) StoreTicketMessage(ctx context.Context, partnerID string, messageID int, ticketID string) error {
	key := fmt.Sprintf("tgbot:ticket_msg:%s:%d", partnerID, messageID)
	return r.redis.Set(ctx, key, ticketID, 0).Err()
}

// GetTicketByMessage retrieves the ticket ID from a Telegram message ID.
func (r *AgentRegistry) GetTicketByMessage(ctx context.Context, partnerID string, messageID int) (string, error) {
	key := fmt.Sprintf("tgbot:ticket_msg:%s:%d", partnerID, messageID)
	return r.redis.Get(ctx, key).Result()
}

// SetActiveTicket sets the active ticket for an agent's Telegram chat.
// All text messages from this chat will be forwarded to this ticket.
func (r *AgentRegistry) SetActiveTicket(ctx context.Context, partnerID string, chatID int64, ticketID string) error {
	key := fmt.Sprintf("tgbot:active_ticket:%s:%d", partnerID, chatID)
	return r.redis.Set(ctx, key, ticketID, 0).Err()
}

// GetActiveTicket returns the active ticket for an agent's Telegram chat.
func (r *AgentRegistry) GetActiveTicket(ctx context.Context, partnerID string, chatID int64) (string, error) {
	key := fmt.Sprintf("tgbot:active_ticket:%s:%d", partnerID, chatID)
	return r.redis.Get(ctx, key).Result()
}

// ClearActiveTicket removes the active ticket for an agent's Telegram chat.
func (r *AgentRegistry) ClearActiveTicket(ctx context.Context, partnerID string, chatID int64) error {
	key := fmt.Sprintf("tgbot:active_ticket:%s:%d", partnerID, chatID)
	return r.redis.Del(ctx, key).Err()
}
