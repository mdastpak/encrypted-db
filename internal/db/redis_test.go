package db

import (
	"context"
	"testing"
	"time"

	"encrypted-db/config"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisService_Success(t *testing.T) {
	originalConfig := config.Config
	defer func() { config.Config = originalConfig }()

	config.Config.Redis.Host = "localhost"
	config.Config.Redis.Port = "6379"
	config.Config.Redis.DB = 1
	config.Config.Redis.ConnectRetries = 1
	config.Config.Redis.ConnectRetryDelayMs = 10

	svc, err := NewRedisService()
	require.NoError(t, err)
	require.NotNil(t, svc)
	defer svc.Close()

	assert.NoError(t, svc.Ping(context.Background()))
}

func TestNewRedisService_FailsWithBadHost(t *testing.T) {
	originalConfig := config.Config
	defer func() { config.Config = originalConfig }()

	config.Config.Redis.Host = "no-such-redis-host-xyz"
	config.Config.Redis.Port = "6379"
	config.Config.Redis.ConnectRetries = 2
	config.Config.Redis.ConnectRetryDelayMs = 10

	svc, err := NewRedisService()
	assert.Error(t, err)
	assert.Nil(t, svc)
}

func TestRedisService_PingNilClient(t *testing.T) {
	svc := &RedisService{}
	err := svc.Ping(context.Background())
	assert.Error(t, err)
}

func TestRedisService_StatsNilClient(t *testing.T) {
	svc := &RedisService{}
	assert.Nil(t, svc.Stats())
}

func TestRedisService_CloseNilClient(t *testing.T) {
	svc := &RedisService{}
	assert.NoError(t, svc.Close())
}

func TestRedisService_StatsAfterConnect(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 1})
	require.NoError(t, client.Ping(context.Background()).Err())
	svc := &RedisService{Client: client}
	defer svc.Close()

	stats := svc.Stats()
	require.NotNil(t, stats)
}

func TestRedisPingWithRetry_Succeeds(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 1})
	defer client.Close()

	err := redisPingWithRetry(client, 3, 10*time.Millisecond)
	assert.NoError(t, err)
}

func TestRedisPingWithRetry_ExhaustsAttempts(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DB: 1})
	defer client.Close()

	start := time.Now()
	err := redisPingWithRetry(client, 2, 20*time.Millisecond)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.GreaterOrEqual(t, elapsed, 20*time.Millisecond)
}

func TestRedisPingWithRetry_NormalizesInvalidArguments(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 1})
	defer client.Close()

	err := redisPingWithRetry(client, 0, 0)
	assert.NoError(t, err)
}
