package system

import (
	"encrypted-db/internal/db"
	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"
)

// SystemHandler struct for system-related endpoints
type SystemHandler struct {
	Postgres        *db.PostgresService
	Redis           *db.RedisService
	RabbitMQService *rabbitmq.RabbitMQService
}

// NewHandler creates a new system handler with injected dependencies
func NewHandler(is *models.InfraServices) *SystemHandler {
	return &SystemHandler{
		Postgres:        is.Postgres,
		Redis:           is.Redis,
		RabbitMQService: is.RabbitMQ,
	}
}
