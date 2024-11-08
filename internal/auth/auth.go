package auth

import (
	"encrypted-db/internal/helpers"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// GenerateAccessToken generates a short-lived JWT (15 minutes) signed with RSA private key
func GenerateAccessToken(username, role string) (string, error) {
	// Set short expiration time for access token
	expirationTime := time.Now().Add(15 * time.Minute).Unix()

	// Define claims for access token
	claims := jwt.MapClaims{
		"username": username,
		"role":     role,
		"exp":      expirationTime, // Short expiration time
		"iss":      "YourAppName",
	}

	// Create and sign the token with RSA private key
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %v", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a long-lived JWT (7 days) signed with RSA private key
func GenerateRefreshToken(username, role string) (string, error) {
	// Set long expiration time for refresh token
	expirationTime := time.Now().Add(7 * 24 * time.Hour).Unix()

	// Define claims for refresh token
	claims := jwt.MapClaims{
		"username": username,
		"role":     role,
		"exp":      expirationTime, // Long expiration time
		"iss":      "YourAppName",
	}

	// Create and sign the token with RSA private key
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %v", err)
	}

	return tokenString, nil
}

// VerifyRSAToken verifies a JWT using the RSA public key
func VerifyRSAToken(tokenString string) (*jwt.MapClaims, error) {
	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	// Handle parsing and validation errors
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// Extract and return claims if the token is valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RefreshTokenHandler refreshes the access token using a valid refresh token
func RefreshTokenHandler(c *gin.Context) {
	// Get refresh token from secure HttpOnly cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		helpers.SendResponse(c, 401, "Refresh token not provided", nil)
		return
	}

	claims := &Claims{}
	token, _ := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	// Validate refresh token
	if err != nil || !token.Valid || claims.ExpiresAt < time.Now().Unix() {
		helpers.SendResponse(c, 401, "Invalid or expired refresh token", nil)
		return
	}

	// Generate new access token
	accessToken, err := GenerateAccessToken(claims.UUID, claims.Role)
	if err != nil {
		helpers.SendResponse(c, 500, "Failed to generate access token", nil)
		return
	}

	// Return new access token
	helpers.SendResponse(c, 200, "Access token refreshed", gin.H{"access_token": accessToken})
}

// SetRefreshTokenCookie sets the refresh token in an HttpOnly cookie
func SetRefreshTokenCookie(c *gin.Context, refreshToken string) {
	c.SetCookie("refresh_token", refreshToken, int(7*24*time.Hour.Seconds()), "/refresh", "", true, true)
}
