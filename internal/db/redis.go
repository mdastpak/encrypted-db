package db

import (
	"context"
	"encrypted-db/config"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
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
