package auth

import (
	"encrypted-db/internal/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

// JWTAdminVerification verifies if the JWT token belongs to an admin
func JWTVerification(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || !verifyAdminToken(token) {
		helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - Admin access required", nil)
		return
	}
	c.Next()
}

// Dummy functions to simulate token verification
func verifyAdminToken(token string) bool {
	// Implement token verification for admin (e.g., check role in token claims)
	return true // Placeholder, implement actual logic here
}

func verifyUserToken(token string) bool {
	// Implement token verification for user (e.g., check role in token claims)
	return true // Placeholder, implement actual logic here
}
