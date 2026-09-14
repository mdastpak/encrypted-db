package auth

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"encrypted-db/config"
	"encrypted-db/internal/helpers"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrKeysNotLoaded     = errors.New("RSA keys not loaded")
	ErrInvalidSigningAlg = errors.New("unexpected signing method")
)

// Claims represents the JWT claims structure.
type Claims struct {
	UUID string `json:"uuid"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken generates a short-lived JWT (15 minutes) signed with RSA private key.
func GenerateAccessToken(uuid, role string) (string, error) {
	if !KeysLoaded() {
		return "", ErrKeysNotLoaded
	}

	expirationTime := time.Now().Add(time.Duration(config.Config.JWT.AccessToken.Expiration) * time.Minute)

	claims := Claims{
		UUID: uuid,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    config.Config.JWT.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(PrivateKey())
	if err != nil {
		log.Printf("Error signing access token: %v", err)
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a long-lived JWT (7 days) signed with RSA private key.
func GenerateRefreshToken(uuid, role string) (string, error) {
	if !KeysLoaded() {
		return "", ErrKeysNotLoaded
	}

	expirationTime := time.Now().Add(time.Duration(config.Config.JWT.RefreshToken.Expiration) * time.Minute)

	claims := Claims{
		UUID: uuid,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    config.Config.JWT.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(PrivateKey())
	if err != nil {
		log.Printf("Error signing refresh token: %v", err)
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// VerifyRSAToken verifies a JWT using the RSA public key.
func VerifyRSAToken(tokenString string) (*Claims, error) {
	if !KeysLoaded() {
		return nil, ErrKeysNotLoaded
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidSigningAlg
		}
		return PublicKey(), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshTokenHandler refreshes the access token using a valid refresh token.
func RefreshTokenHandler(c *gin.Context) {
	if !KeysLoaded() {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to load RSA keys", nil)
		return
	}

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		helpers.SendResponse(c, http.StatusUnauthorized, "Refresh token not provided", nil)
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidSigningAlg
		}
		return PublicKey(), nil
	})

	if err != nil || !token.Valid {
		helpers.SendResponse(c, http.StatusUnauthorized, "Invalid or expired refresh token", nil)
		return
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		helpers.SendResponse(c, http.StatusUnauthorized, "Expired refresh token", nil)
		return
	}

	accessToken, err := GenerateAccessToken(claims.UUID, claims.Role)
	if err != nil {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to generate access token", nil)
		return
	}

	helpers.SendResponse(c, http.StatusOK, "Access token refreshed", gin.H{"access_token": accessToken})
}

// SetRefreshTokenCookie sets the refresh token in an HttpOnly cookie.
func SetRefreshTokenCookie(c *gin.Context, refreshToken string) {
	maxAge := config.Config.JWT.RefreshToken.Expiration * 60
	c.SetCookie("refresh_token", refreshToken, maxAge, "/", "", true, true)
}

// VerifyAdminToken verifies if the token is valid and has admin role.
func VerifyAdminToken(tokenString string) bool {
	claims, err := VerifyRSAToken(tokenString)
	if err != nil {
		return false
	}
	return claims.Role == "admin"
}

// VerifyUserToken verifies if the token is valid and has user role.
func VerifyUserToken(tokenString string) bool {
	claims, err := VerifyRSAToken(tokenString)
	if err != nil {
		return false
	}
	return claims.Role == "user"
}
