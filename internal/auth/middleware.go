package auth

import (
	"net/http"

	"encrypted-db/internal/helpers"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware validates tokens and checks if they are blacklisted.
func JWTMiddleware(tb *TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - Token not provided", nil)
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, ErrInvalidSigningAlg
			}
			return PublicKey(), nil
		})

		if err != nil || !token.Valid {
			helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - Invalid token", nil)
			c.Abort()
			return
		}

		if tb != nil {
			isBlacklisted, err := tb.IsTokenBlacklisted(c.Request.Context(), tokenString)
			if err != nil {
				helpers.SendResponse(c, http.StatusInternalServerError, "Error checking blacklist", nil)
				c.Abort()
				return
			}
			if isBlacklisted {
				helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - Token is blacklisted", nil)
				c.Abort()
				return
			}
		}

		c.Set("claims", claims)
		c.Next()
	}
}

// GetClaims extracts claims from the context.
func GetClaims(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	casted, ok := claims.(*Claims)
	return casted, ok
}

// JWTAdminVerification verifies if the JWT token belongs to an admin.
func JWTAdminVerification(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || !VerifyAdminToken(token) {
		helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - Admin access required", nil)
		c.Abort()
		return
	}
	c.Next()
}

// JWTUserVerification verifies if the JWT token belongs to a user.
func JWTUserVerification(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" || !VerifyUserToken(token) {
		helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - User access required", nil)
		c.Abort()
		return
	}
	c.Next()
}
