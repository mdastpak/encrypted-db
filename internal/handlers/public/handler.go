package public

import (
	"encrypted-db/internal/db"
	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"
)

// PublicHandler struct to hold dependencies for public routes
type PublicHandler struct {
	PostgresDB      *db.PostgresService
	RedisClient     *db.RedisService
	RedisIndex      string // Redis index for base definitions
	RabbitMQService *rabbitmq.RabbitMQService
}

// NewHandler function to initialize PublicHandler with dependencies
func NewHandler(is *models.InfraServices) *PublicHandler {
	return &PublicHandler{
		PostgresDB:      is.Postgres,
		RedisClient:     is.Redis,
		RedisIndex:      "base_definitions", // Setting index for base definitions
		RabbitMQService: is.RabbitMQ,
	}
}

// CurrencyResponse represents the custom output format for each currency
type CurrencyResponse struct {
	HK     string              `json:"hk"`
	Info   models.CurrencyInfo `json:"info"`
	Status string              `json:"status"`
}
