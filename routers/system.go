package routers

import (
	"encrypted-db/services"

	"github.com/gin-gonic/gin"
)

func SystemRoutes(router *gin.RouterGroup, services *services.Services) {
	// systemGroup := router.Group("/system")
	// {
	// 	systemGroup.GET("/status", system.StatusHandler(services))
	// 	systemGroup.GET("/health", system.HealthCheckHandler(services))
	// }
}
