package auth

import (
	"net/http"
	"strings"
	"time"

	"encrypted-db/internal/helpers"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func LogoutHandler(c *gin.Context, tb *TokenBlacklist) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		helpers.SendResponse(c, http.StatusUnauthorized, "Unauthorized - Token not provided", nil)
		return
	}
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
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
		return
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	} else {
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	ctx := c.Request.Context()
	if err := tb.AddTokenToBlacklist(ctx, tokenString, expiresAt); err != nil {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to logout", nil)
		return
	}

	helpers.SendResponse(c, http.StatusOK, "Logged out successfully", nil)
}
