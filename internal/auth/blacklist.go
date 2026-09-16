package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/go-redis/redis/v8"
)

type TokenBlacklist struct {
	RedisClient *redis.Client
	Prefix      string
}

func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{
		RedisClient: client,
		Prefix:      "blacklist:token:",
	}
}

func (tb *TokenBlacklist) tokenKey(token string) string {
	hash := sha256.Sum256([]byte(token))
	return tb.Prefix + hex.EncodeToString(hash[:])
}

func (tb *TokenBlacklist) AddTokenToBlacklist(ctx context.Context, token string, expiresAt time.Time) error {
	if tb.RedisClient == nil {
		return nil // No-op if no Redis client
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	key := tb.tokenKey(token)
	return tb.RedisClient.Set(ctx, key, "1", ttl).Err()
}

func (tb *TokenBlacklist) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	if tb.RedisClient == nil {
		return false, nil // No-op if no Redis client
	}
	key := tb.tokenKey(token)
	val, err := tb.RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}
