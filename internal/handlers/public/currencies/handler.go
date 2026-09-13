package currencies

import (
	"encrypted-db/internal/handlers/public"
	"encrypted-db/services"
)

// CurrenciesPublicHandler struct to hold dependencies for public routes
type CurrenciesPublicHandler struct {
	Handler    *public.Handler // Added PublicHandler dependency
	RedisIndex string          // Redis index for base definitions
}

// NewHandler function to initialize CurrenciesPublicHandler with dependencies
func CurrenciesNewHandler(services *services.Services) *CurrenciesPublicHandler {
	return &CurrenciesPublicHandler{
		RedisIndex: "base_definitions", // Setting index for base definitions
		Handler: &public.Handler{
			Postgres:        services.Postgres,
			Redis:           services.Redis,
			RabbitMQService: services.RabbitMQ,
		},
	}
}
