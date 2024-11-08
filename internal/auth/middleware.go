package auth

import (
	"encrypted-db/internal/helpers"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// JWTMiddleware validates tokens and checks if they are blacklisted
func JWTMiddleware(tb *TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the token from Authorization header
		tokenString := c.GetHeader("Authorization")

		// Parse the JWT token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return publicKey, nil
		})

		// Check if token is valid
		if err != nil || !token.Valid {
			helpers.SendResponse(c, 401, "Unauthorized - Invalid token", nil)
			return
		}

		// Check if token is blacklisted
		isBlacklisted, err := tb.IsTokenBlacklisted(tokenString)
		if err != nil {
			helpers.SendResponse(c, 500, "Error checking blacklist", nil)
			return
		}
		if isBlacklisted {
			helpers.SendResponse(c, 401, "Unauthorized - Token is blacklisted", nil)
			return
		}

		// Token is valid and not blacklisted; proceed to the next handler
		c.Next()
	}
}
