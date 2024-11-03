package system

import (
	"encrypted-db/internal/db"
	"encrypted-db/internal/rabbitmq"
)

// SystemHandler struct for system-related endpoints
type SystemHandler struct {
	Postgres        *db.PostgresService
	Redis           *db.RedisService
	RabbitMQService *rabbitmq.RabbitMQService
}

// NewHandler creates a new system handler with injected dependencies
func NewHandler(postgres *db.PostgresService, redis *db.RedisService, rabbitMQ *rabbitmq.RabbitMQService) *SystemHandler {
	return &SystemHandler{
		Postgres:        postgres,
		Redis:           redis,
		RabbitMQService: rabbitMQ,
	}
}
