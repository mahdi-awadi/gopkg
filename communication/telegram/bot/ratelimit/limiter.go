package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a sliding window rate limiter per user.
type RateLimiter struct {
	redis     *redis.Client
	maxPerMin int
}

func NewRateLimiter(rdb *redis.Client, maxPerMin int) *RateLimiter {
	if maxPerMin <= 0 {
		maxPerMin = 20
	}
	return &RateLimiter{redis: rdb, maxPerMin: maxPerMin}
}

// Allow returns true if the user is within the rate limit.
func (rl *RateLimiter) Allow(ctx context.Context, partnerID string, telegramID int64) (bool, error) {
	key := fmt.Sprintf("tgbot:rl:%s:%d", partnerID, telegramID)
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	pipe := rl.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMilli()), Member: now.UnixMilli()})
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, 2*time.Minute)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, err // Fail open on Redis errors
	}

	return countCmd.Val() <= int64(rl.maxPerMin), nil
}
