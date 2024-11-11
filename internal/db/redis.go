package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encrypted-db/config"
	"fmt"
	"log"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RedisService struct {
	Client *redis.Client
	Ctx    context.Context
}

// NewRedisService sets up the Redis client and context for injection
func NewRedisService() *RedisService {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Config.Redis.Host, config.Config.Redis.Port),
		Password: config.Config.Redis.Password,
		DB:       config.Config.Redis.DB,
	})

	if _, err := client.Ping(ctx).Result(); err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}

	return &RedisService{
		Client: client,
		Ctx:    ctx,
	}
}

// Close closes the Redis connection
func (r *RedisService) Close() {
	if err := r.Client.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}
}

// GenerateRedisKey generates a Redis key by hashing concatenated input parameters with a separator
// Example usage
// fmt.Println(GenerateRedisKey(":", "operation", "contact", "otp"))   // Default separator `:`
// fmt.Println(GenerateRedisKey("-", "param1", "param2", "param3"))    // Custom separator `-`
// fmt.Println(GenerateRedisKey("", "single", "value", "test"))        // No separator provided, uses default `:`
// fmt.Println(GenerateRedisKey(":", ""))                              // Empty value, should use default UUID
// fmt.Println(GenerateRedisKey(":"))                                  // No values, should use default UUID
func (r *RedisService) GenerateRedisKey(values ...string) string {
	// Set default separator
	separator := ":"

	// Check if values is empty, if so generate a new UUID
	if len(values) == 0 {
		newUUID := uuid.New().String()
		values = []string{newUUID}
	}

	// Concatenate all values with the separator
	combined := strings.Join(values, separator)

	// Generate SHA-256 hash of the combined string
	hash := sha256.Sum256([]byte(combined))

	// Return the hash as a hexadecimal string
	return hex.EncodeToString(hash[:])
}
