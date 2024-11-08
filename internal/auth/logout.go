package auth

import (
	"encrypted-db/internal/helpers"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// LogoutHandler adds the token to blacklist upon user logout
func LogoutHandler(c *gin.Context, tb *TokenBlacklist) {
	// Retrieve the token from Authorization header
	tokenString := c.GetHeader("Authorization")

	// Parse the token to get claims and expiration
	claims := &Claims{}

	if tokenString == "" {
		helpers.SendResponse(c, 401, "Unauthorized - Token not provided", nil)
		return

	}

	// Parse token and validate claims
	token, _ := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if token == nil {
		helpers.SendResponse(c, 401, "Unauthorized - Invalid token", nil)
		return
	}

	// Get expiration time from token claims
	expiresAt := time.Unix(claims.ExpiresAt, 0)

	// Add token to blacklist
	err := tb.AddTokenToBlacklist(tokenString, expiresAt)
	if err != nil {
		helpers.SendResponse(c, 500, "Failed to logout", nil)
		return
	}

	// Successfully logged out
	helpers.SendResponse(c, 200, "Logged out successfully", nil)
}
