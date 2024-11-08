package auth

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type TokenBlacklist struct {
	RedisClient *redis.Client
}

// AddTokenToBlacklist adds a token to Redis blacklist with expiration time
func (tb *TokenBlacklist) AddTokenToBlacklist(token string, expiresAt time.Time) error {
	// Calculate time-to-live (TTL) for the token in seconds
	ttl := time.Until(expiresAt)

	// Store the token in Redis with expiration time
	return tb.RedisClient.Set(ctx, token, true, ttl).Err()
}

// IsTokenBlacklisted checks if a token is blacklisted in Redis
func (tb *TokenBlacklist) IsTokenBlacklisted(token string) (bool, error) {
	// Check if token exists in Redis
	val, err := tb.RedisClient.Get(ctx, token).Result()
	if err == redis.Nil {
		// Token not found in blacklist
		return false, nil
	} else if err != nil {
		// Some other Redis error occurred
		return false, err
	}
	// Token found in blacklist
	return val == "true", nil
}
