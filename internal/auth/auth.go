package auth

import (
	"encrypted-db/config"
	"encrypted-db/internal/helpers"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// GenerateAccessToken generates a short-lived JWT (15 minutes) signed with RSA private key
func GenerateAccessToken(uuid, role string) (string, error) {
	log.Println("Generating access token...")

	if err := LoadKeys(); err != nil {
		log.Printf("Error loading RSA keys: %v\n", err)
		return "", fmt.Errorf("failed to load RSA keys: %v", err)
	}

	// Set short expiration time for access token
	expirationTime := time.Now().Add(time.Duration(config.Config.JWT.AccessToken.Expiration) * time.Minute).Unix()
	log.Printf("Access token expiration time set to: %v\n", expirationTime)

	// Define claims for access token
	claims := jwt.MapClaims{
		"uuid": uuid,
		"role": role,
		"exp":  expirationTime,
		"iss":  config.Config.JWT.Issuer,
	}
	log.Printf("Access token claims: uuid=%s, role=%s, exp=%v, iss=%s\n", uuid, role, expirationTime, config.Config.JWT.Issuer)

	if privateKey == nil {
		log.Println("Error: privateKey is nil. Ensure it is loaded correctly.")
		return "", fmt.Errorf("private key is not loaded")
	}

	// Create and sign the token with RSA private key
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		log.Printf("Error signing access token: %v\n", err)
		return "", fmt.Errorf("failed to sign access token: %v", err)
	}

	log.Println("Access token generated successfully")
	return tokenString, nil
}

// GenerateRefreshToken generates a long-lived JWT (7 days) signed with RSA private key
func GenerateRefreshToken(uuid, role string) (string, error) {
	log.Println("Generating refresh token...")

	// Set long expiration time for refresh token
	expirationTime := time.Now().Add(time.Duration(config.Config.JWT.RefreshToken.Expiration) * time.Minute).Unix()
	log.Printf("Refresh token expiration time set to: %v\n", expirationTime)

	// Define claims for refresh token
	claims := jwt.MapClaims{
		"uuid": uuid,
		"role": role,
		"exp":  expirationTime,
		"iss":  config.Config.JWT.Issuer,
	}
	log.Printf("Refresh token claims: uuid=%s, role=%s, exp=%v, iss=%s\n", uuid, role, expirationTime, config.Config.JWT.Issuer)

	// Create and sign the token with RSA private key
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		log.Printf("Error signing refresh token: %v\n", err)
		return "", fmt.Errorf("failed to sign refresh token: %v", err)
	}

	log.Println("Refresh token generated successfully")
	return tokenString, nil
}

// VerifyRSAToken verifies a JWT using the RSA public key
func VerifyRSAToken(tokenString string) (*jwt.MapClaims, error) {
	log.Println("Starting token verification process...")

	if err := LoadKeys(); err != nil {
		log.Printf("Error loading RSA keys: %v\n", err)
		return nil, fmt.Errorf("failed to load RSA keys: %v", err)
	}

	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		log.Println("Parsing token and validating signing method...")

		// Ensure the signing method is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			log.Printf("Unexpected signing method: %v\n", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		log.Println("Signing method is valid. Using RSA public key for verification.")
		return publicKey, nil
	})

	// Handle parsing and validation errors
	if err != nil {
		log.Printf("Failed to parse token: %v\n", err)
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	log.Println("Token parsed successfully. Extracting claims...")

	// Extract and return claims if the token is valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		log.Println("Token is valid. Claims extracted successfully.")
		return &claims, nil
	}

	log.Println("Token is invalid or claims extraction failed.")
	return nil, fmt.Errorf("invalid token")
}

// RefreshTokenHandler refreshes the access token using a valid refresh token
func RefreshTokenHandler(c *gin.Context) {
	log.Println("Starting refresh token handler...")

	if err := LoadKeys(); err != nil {
		log.Printf("Error loading RSA keys: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to load RSA keys", nil)
	}

	// Get refresh token from secure HttpOnly cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		log.Println("Refresh token not provided in cookie.")
		helpers.SendResponse(c, http.StatusUnauthorized, "Refresh token not provided", nil)
		return
	}
	log.Println("Refresh token retrieved from cookie.")

	// Parse and validate the refresh token
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		log.Println("Validating signing method for refresh token...")

		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			log.Printf("Unexpected signing method: %v\n", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		log.Println("Signing method is valid. Using RSA public key for verification.")
		return publicKey, nil
	})

	// Handle token parsing and validation errors
	if err != nil || !token.Valid {
		log.Printf("Invalid or expired refresh token: %v\n", err)
		helpers.SendResponse(c, http.StatusUnauthorized, "Invalid or expired refresh token", nil)
		return
	}
	log.Println("Refresh token is valid.")

	// Extract claims and check expiration
	uuid, uuidOk := claims["uuid"].(string)
	role, roleOk := claims["role"].(string)
	exp, expOk := claims["exp"].(float64)

	if !uuidOk || !roleOk || !expOk {
		log.Println("Error: Missing or invalid claims in refresh token.")
		helpers.SendResponse(c, http.StatusUnauthorized, "Invalid refresh token claims", nil)
		return
	}

	// Check token expiration
	if int64(exp) < time.Now().Unix() {
		log.Println("Refresh token has expired.")
		helpers.SendResponse(c, http.StatusUnauthorized, "Expired refresh token", nil)
		return
	}

	log.Printf("Claims extracted: uuid=%s, role=%s, exp=%v\n", uuid, role, exp)

	// Generate new access token
	accessToken, err := GenerateAccessToken(uuid, role)
	if err != nil {
		log.Printf("Failed to generate new access token: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to generate access token", nil)
		return
	}

	log.Println("New access token generated successfully.")

	// Return the new access token
	helpers.SendResponse(c, http.StatusOK, "Access token refreshed", gin.H{"access_token": accessToken})
}

// SetRefreshTokenCookie sets the refresh token in an HttpOnly cookie
func SetRefreshTokenCookie(c *gin.Context, refreshToken string) {
	// c.SetCookie("refresh_token", refreshToken, int(7*24*time.Hour.Seconds()), "/refresh", "", true, true)
	c.SetCookie("refresh_token", refreshToken, config.Config.JWT.RefreshToken.Expiration*60, "/user/refresh", "", true, true)
}
