package user

import (
	"encrypted-db/internal/db"
	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"
)

// UserHandler struct to hold dependencies for User routes
type UserHandler struct {
	PostgresDB      *db.PostgresService
	RedisClient     *db.RedisService
	RedisIndex      string // Redis index for base definitions
	RabbitMQService *rabbitmq.RabbitMQService
}

// NewHandler function to initialize CurrenciesUserHandler with dependencies
func NewHandler(is *models.InfraServices) *UserHandler {
	return &UserHandler{
		PostgresDB:      is.Postgres,
		RedisClient:     is.Redis,
		RabbitMQService: is.RabbitMQ,
	}
}
