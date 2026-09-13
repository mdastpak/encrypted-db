package currencies

import (
	"encrypted-db/internal/handlers/admin"
	"encrypted-db/services"
)

// CurrenciesAdminHandler struct to hold dependencies for admin routes
type CurrenciesAdminHandler struct {
	Handler    *admin.Handler // Added AdminHandler dependency
	RedisIndex string         // Redis index for base definitions
}

// NewHandler function to initialize CurrenciesAdminHandler with dependencies
func CurrenciesNewHandler(services *services.Services) *CurrenciesAdminHandler {
	return &CurrenciesAdminHandler{
		RedisIndex: "base_definitions", // Setting index for base definitions
		Handler: &admin.Handler{
			Postgres:        services.Postgres,
			Redis:           services.Redis,
			RabbitMQService: services.RabbitMQ,
		},
	}
}

// // CloseHandler closes the context when the handler is no longer needed
// func (h *CurrenciesAdminHandler) CloseHandler() {
// 	h.Handler.CloseHandler() // Calls the CloseHandler method from the AdminHandler
// }
