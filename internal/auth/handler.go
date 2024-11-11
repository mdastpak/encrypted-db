package auth

import (
	"encrypted-db/internal/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

// JWTAdminVerification verifies if the JWT token belongs to an admin
func JWTAdminVerification(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || !verifyAdminToken(token) {
		helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - Admin access required", nil)
		return
	}
	c.Next()
}

// JWTAdminVerification verifies if the JWT token belongs to an user
func JWTUserVerification(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || !verifyUserToken(token) {
		helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - User access required", nil)
		return
	}
	c.Next()
}

// Dummy functions to simulate token verification
func verifyAdminToken(token string) bool {
	// Implement token verification for admin (e.g., check role in token claims)
	return true // Placeholder, implement actual logic here
}

// verifyUserToken verifies if the token is valid and has a "user" role
func verifyUserToken(token string) bool {
	// Verify token and get claims
	claims, err := VerifyRSAToken(token)
	if err != nil {
		return false
	}

	// Check if claims contain role and if it's "user"
	role, ok := (*claims)["role"].(string)
	if !ok || role != "user" {
		return false
	}

	return true
}
