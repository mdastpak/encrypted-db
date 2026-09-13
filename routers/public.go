package routers

import (
	"encrypted-db/internal/handlers/public/currencies"
	"encrypted-db/services"

	"github.com/gin-gonic/gin"
)

func PublicRoutes(router *gin.RouterGroup, services *services.Services) {
	currenciesRouter := router.Group("/currencies")
	{
		currenciesHandler := currencies.CurrenciesNewHandler(services)
		defer currenciesHandler.Handler.CloseHandler()

		currenciesRouter.GET("/", currenciesHandler.GetActiveCurrencies)
		currenciesRouter.GET("/:hk", currenciesHandler.GetCurrencyByHK)
	}
}

// func Router(router *gin.RouterGroup, postgresService *db.PostgresService, redisService *db.RedisService, rabbitMQService *rabbitmq.RabbitMQService) {

// }
