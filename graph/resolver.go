package graph

import (
	"encrypted-db/graph/model"
	"encrypted-db/internal/db"
	"sync"
)

type Resolver struct {
	Postgres *db.PostgresService
	Redis    *db.RedisService

	// For managing subscriptions
	MessageSubscribers map[string]chan *model.Message
	mu                 sync.Mutex
}

func NewResolver(postgres *db.PostgresService, redis *db.RedisService) *Resolver {
	return &Resolver{
		Postgres:           postgres,
		Redis:              redis,
		MessageSubscribers: make(map[string]chan *model.Message),
	}
}
