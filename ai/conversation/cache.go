package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Match the existing core-service/services/ai/cache.go behaviour exactly
// so dropping that file in Task 12 is a behavioural no-op for chat users.
const (
	ConversationTTL      = time.Hour
	ConversationMaxItems = 20
)

// Cache is the interface callers depend on. Backed by Redis in
// production, by miniredis (or any go-redis-compatible) in tests.
type Cache interface {
	Get(ctx context.Context, key string) ([]Message, error)
	Append(ctx context.Context, key string, msg Message) error
	Recent(ctx context.Context, key string, n int) ([]Message, error)
	Clear(ctx context.Context, key string) error
}

// redisCache is the Redis-backed implementation.
type redisCache struct {
	client *redis.Client
}

// NewRedisCache builds a redis-backed Cache.
func NewRedisCache(client *redis.Client) Cache {
	return &redisCache{client: client}
}

func (c *redisCache) Get(ctx context.Context, key string) ([]Message, error) {
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	var msgs []Message
	if err := json.Unmarshal([]byte(data), &msgs); err != nil {
		return nil, fmt.Errorf("unmarshal conversation: %w", err)
	}
	return msgs, nil
}

func (c *redisCache) Append(ctx context.Context, key string, msg Message) error {
	msgs, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	msgs = append(msgs, msg)
	if len(msgs) > ConversationMaxItems {
		msgs = msgs[len(msgs)-ConversationMaxItems:]
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("marshal conversation: %w", err)
	}
	if err := c.client.Set(ctx, key, data, ConversationTTL).Err(); err != nil {
		return fmt.Errorf("save conversation: %w", err)
	}
	return nil
}

func (c *redisCache) Recent(ctx context.Context, key string, n int) ([]Message, error) {
	msgs, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if n <= 0 || len(msgs) <= n {
		return msgs, nil
	}
	return msgs[len(msgs)-n:], nil
}

func (c *redisCache) Clear(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("clear conversation: %w", err)
	}
	return nil
}
