package system

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// PingPong responds with the server time based on the specified timezone
func (h *SystemHandler) PingPongHandler(c *gin.Context) {
	timezone := c.Query("timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	serverTime := time.Now().In(loc).Format(time.RFC3339)

	c.JSON(http.StatusOK, gin.H{
		"event":     "pong",
		"timestamp": serverTime,
	})
}
