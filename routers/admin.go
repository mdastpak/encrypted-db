package routers

import (
	"encrypted-db/internal/handlers/admin/currencies"
	"encrypted-db/services"

	"github.com/gin-gonic/gin"
)

func AdminRoutes(router *gin.RouterGroup, services *services.Services) {
	currenciesGroup := router.Group("/currencies")
	{
		adminCurrenciesHandler := currencies.CurrenciesNewHandler(services)
		defer adminCurrenciesHandler.Handler.CloseHandler()

		currenciesGroup.POST("/", adminCurrenciesHandler.CreateCurrency)
		currenciesGroup.PUT("/:hk", adminCurrenciesHandler.UpdateCurrency)
		currenciesGroup.DELETE("/:hk", adminCurrenciesHandler.DeleteCurrency)
	}

	// adminGroup.GET("/someRoute", admin.SomeHandler(services))
	// adminGroup.POST("/anotherRoute", admin.AnotherHandler(services))
}
