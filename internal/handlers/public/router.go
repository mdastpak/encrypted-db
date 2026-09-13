package public

// import (
// 	"encrypted-db/internal/db"
// 	"encrypted-db/internal/handlers/public/currencies"
// 	"encrypted-db/internal/rabbitmq"

// 	"github.com/gin-gonic/gin"
// )

// func Router(router *gin.RouterGroup, postgresService *db.PostgresService, redisService *db.RedisService, rabbitMQService *rabbitmq.RabbitMQService) {
// 	currenciesRouter := router.Group("/currencies")
// 	{
// 		currenciesHandler := currencies.CurrenciesNewHandler(postgresService, redisService, rabbitMQService)
// 		defer currenciesHandler.Handler.CloseHandler()

// 		currenciesRouter.GET("/", currenciesHandler.GetActiveCurrencies)
// 		currenciesRouter.GET("/:hk", currenciesHandler.GetCurrencyByHK)
// 	}
// }
