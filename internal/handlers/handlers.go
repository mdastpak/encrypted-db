// internal/handlers/handlers.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler for the admin route
func AdminHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Welcome, Admin!"})
}

// Handler for the user route
func UserHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Welcome, User!"})
}
