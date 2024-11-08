package admin

import (
	"encrypted-db/internal/db"
	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"
)

// AdminHandler struct to hold dependencies for admin routes
type AdminHandler struct {
	PostgresDB      *db.PostgresService
	RedisClient     *db.RedisService
	RedisIndex      string // Redis index for base definitions
	RabbitMQService *rabbitmq.RabbitMQService
}

// NewHandler function to initialize CurrenciesAdminHandler with dependencies
func NewHandler(is *models.InfraServices) *AdminHandler {
	return &AdminHandler{
		PostgresDB:      is.Postgres,
		RedisClient:     is.Redis,
		RedisIndex:      "base_definitions", // Setting index for base definitions
		RabbitMQService: is.RabbitMQ,
	}
}
