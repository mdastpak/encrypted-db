package routers

import (
	"encrypted-db/services"

	"github.com/gin-gonic/gin"
)

func SocketRoutes(router *gin.RouterGroup, services *services.Services) {
	// socketGroup := router.Group("/ws")
	// {
	// 	socketGroup.GET("/", socket.SomeSocketHandler(services))
	// }
}
