package public

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"encrypted-db/config"
	"encrypted-db/internal/auth"
	"encrypted-db/internal/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/net/context"
)

// OTPRequest struct for OTP request
type OTPRequest struct {
	Username string `json:"username"`
}

// VerifyOTPRequest struct for OTP verification request
type VerifyOTPRequest struct {
	UUID string `json:"uuid"`
	OTP  string `json:"otp"`
}

// User represents a user in the database
type User struct {
	ID        int
	HK        uuid.UUID
	Status    string
	Info      map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime // Use sql.NullTime for nullable datetime column
}

// RequestOTP handles OTP generation and storage in Redis
func (h *PublicHandler) RequestOTP(c *gin.Context) {
	var req OTPRequest

	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid request data: Username, Email or Mobile is required", nil)
		return
	}

	otpCode := GenerateOTP(config.Config.OTP.AUTH.Length)
	userUUID := uuid.New().String()

	contactType, maskedContact := IdentifyInputType(req.Username)
	if contactType == "" {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid contact information", nil)
		return
	}

	redisKey := h.RedisClient.GenerateRedisKey(config.Config.OTP.AUTH.Name, userUUID, otpCode)
	authVal, e1 := EncryptWithAES(req.Username, userUUID)
	if e1 != nil || authVal == "" {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to encrypt OTP data", e1)
		return
	}

	otpData := map[string]interface{}{
		"auth": authVal,
		"uuid": userUUID,
	}

	otpJSON, _ := json.Marshal(otpData)
	ttl := time.Duration(config.Config.OTP.AUTH.TTL) * time.Second
	err := h.RedisClient.Client.Set(context.Background(), redisKey, otpJSON, ttl).Err()
	if err != nil {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to store OTP", nil)
		return
	}

	fmt.Printf("OTP Code for %s: %s: %s\n", contactType, req.Username, otpCode)
	helpers.SendResponse(c, http.StatusOK, "OTP sent to your contact", gin.H{"type": contactType, "username": maskedContact, "uuid": userUUID})
}

// VerifyOTP handles OTP verification and user validation
func (h *PublicHandler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest

	reqUUID := c.Param("uuid")
	if err := c.ShouldBindJSON(&req); err != nil || reqUUID == "" || req.OTP == "" {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid request data: UUID and OTP are required", nil)
		return
	}

	redisKey := h.RedisClient.GenerateRedisKey(config.Config.OTP.AUTH.Name, reqUUID, req.OTP)
	otpData, err := h.RedisClient.Client.Get(context.Background(), redisKey).Result()
	if err != nil {
		h.incrementBlacklistAttempts(reqUUID)
		helpers.SendResponse(c, http.StatusUnauthorized, "Invalid or expired OTP code", nil)
		return
	}

	var otpInfo map[string]interface{}
	if err := json.Unmarshal([]byte(otpData), &otpInfo); err != nil {
		helpers.SendResponse(c, http.StatusUnauthorized, "Failed to parse OTP data", nil)
		return
	}

	if otpInfo["uuid"] != reqUUID {
		helpers.SendResponse(c, http.StatusUnauthorized, "Invalid UUID", nil)
		return
	}

	contact, e1 := DecryptWithAES(otpInfo["auth"].(string), reqUUID)
	if e1 != nil || contact == "" {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to decrypt OTP data", nil)
		return
	}

	h.RedisClient.Client.Del(context.Background(), redisKey)
	contactType, _ := IdentifyInputType(contact)
	if contactType == "" {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid contact information", nil)
		return
	}

	// h.checkOrCreateUser(c, contactType, contact)

	userHK, err := h.checkOrCreateUser(contactType, contact)

	if err != nil || userHK == uuid.Nil {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to create user", nil)
		return
	}
	log.Printf("User HK: %s\n", userHK.String())

	accessToken, err := auth.GenerateAccessToken(userHK.String(), "user")
	if err != nil {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to generate access token", nil)
		return
	}
	refreshToken, err := auth.GenerateRefreshToken(userHK.String(), "user")
	if err != nil {
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to generate refresh token", nil)
		return
	}

	auth.SetRefreshTokenCookie(c, refreshToken)

	helpers.SendResponse(c, http.StatusOK, "OTP verified successfully", gin.H{"access_token": accessToken, "refresh_token": refreshToken})

}

// checkOrCreateUser checks if a user exists or creates a new user if not
// Returns HTTP status code and a message as string
func (h *PublicHandler) checkOrCreateUser(contactType, contact string) (uuid.UUID, error) {
	log.Println("Checking if user exists or creating a new user")

	// Construct the query with dynamic JSON extraction
	query := fmt.Sprintf(`SELECT hk, status FROM users WHERE info->'contact'->>'%s' = $1`, contactType)
	var user User
	// var infoData []byte // Define infoData as []byte to hold JSON data from `info` column

	// Try to find the user in the database
	err := h.PostgresDB.DB.QueryRow(query, contact).Scan(&user.HK, &user.Status)
	if err == sql.ErrNoRows {
		// If user does not exist, create a new user
		log.Println("User does not exist, creating new user")
		return h.createUser(contactType, contact)
	} else if err != nil {
		// Database error occurred
		log.Printf("Database error: %v\n", err)
		return uuid.Nil, fmt.Errorf("database error. please try again")
	}

	if user.Status == "disabled" {
		log.Printf("User is disabled. Status: %s\n", user.Status)
		return uuid.Nil, fmt.Errorf("user is disabled. please contact support")
	}
	if user.Status == "deleted" {
		log.Printf("User is deleted. Status: %s\n", user.Status)
		return uuid.Nil, fmt.Errorf("user is deleted. please contact support")
	}
	if user.Status == "suspend" {
		log.Printf("User is suspended. Status: %s\n", user.Status)
		return uuid.Nil, fmt.Errorf("user is suspended. please contact support")
	}

	// // Decode JSON data from `info` into the `Info` field
	// if err := json.Unmarshal(infoData, &user.Info); err != nil {
	// 	log.Printf("Error unmarshalling user info data: %v\n", err)
	// 	return uuid.Nil, fmt.Errorf("failed to fetch user info. please try again")
	// }

	// User already exists
	log.Printf("User already exists with ID: %s\n", user.HK.String())
	return user.HK, nil
}

// createUser inserts a new user in the database
// Returns HTTP status code and a message as string
func (h *PublicHandler) createUser(contactType, contact string) (uuid.UUID, error) {
	// Prepare the user info in JSON format
	info := map[string]interface{}{
		"contact": map[string]interface{}{
			contactType: contact,
		},
	}
	infoData, _ := json.Marshal(info)

	// Insert the new user into the database
	query := `INSERT INTO users (info, status) VALUES ($1, 'approved') RETURNING hk`
	var userHK uuid.UUID
	err := h.PostgresDB.DB.QueryRow(query, infoData).Scan(&userHK)
	if err != nil {
		log.Printf("Failed to insert new user: %v\n", err)
		return uuid.Nil, fmt.Errorf("failed to create user. please try again")
	}

	// User created successfully
	log.Printf("New user created successfully with ID: %s\n", userHK)
	return userHK, nil
}

// IdentifyInputType checks the format of the input, returns the type (email, mobile, or username), and a masked version of the input
func IdentifyInputType(input string) (string, string) {
	// Regex patterns
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	mobileRegex := `^09\d{9}$`
	usernameRegex := `^[a-zA-Z0-9_]{6,20}$` // Only checks for allowed characters and length

	// Helper function to mask a substring
	mask := func(visibleStart, visibleEnd int, str string) string {
		if len(str) <= visibleStart+visibleEnd {
			return str // No masking if input is too short
		}
		maskedPart := strings.Repeat("*", 5)
		return str[:visibleStart] + maskedPart + str[len(str)-visibleEnd:]
	}

	// Check if input matches email format
	if matched, _ := regexp.MatchString(emailRegex, input); matched {
		// Mask all except the first 3 characters and domain part
		atIndex := strings.Index(input, "@")
		if atIndex > 1 {
			return "email", mask(3, len(input)-atIndex, input)
		}
	}

	// Check if input matches mobile format
	if matched, _ := regexp.MatchString(mobileRegex, input); matched {
		// Mask all except the first 3 and last 3 characters for mobile
		return "mobile", mask(3, 3, input)
	}

	// Check if input matches username format
	if matched, _ := regexp.MatchString(usernameRegex, input); matched {
		// Additional checks for username:
		// - Should not start with a digit or underscore
		// - Should not have consecutive underscores
		// - Should contain at least one alphanumeric character between underscores

		// Check if the first character is a letter (not a digit or underscore)
		firstChar := rune(input[0])
		if unicode.IsDigit(firstChar) || firstChar == '_' {
			return "", ""
		}

		// Check for consecutive underscores
		consecutiveUnderscores := regexp.MustCompile(`__`)
		if consecutiveUnderscores.MatchString(input) {
			return "", ""
		}

		// Check if there is at least one alphanumeric character
		containsAlphanumeric := regexp.MustCompile(`[a-zA-Z0-9]`)
		if containsAlphanumeric.MatchString(input) {
			// Mask all except the first 2 and last 3 characters for username
			return "username", mask(2, 3, input)
		}
	}

	// Return "" if none of the patterns match
	return "", ""
}

// Increment the blacklist count if OTP fails
func (h *PublicHandler) incrementBlacklistAttempts(uuid string) {
	blacklistKey := "uuid_blacklist:" + uuid
	attempts, _ := h.RedisClient.Client.Get(context.Background(), blacklistKey).Int()
	attempts++
	h.RedisClient.Client.Set(context.Background(), blacklistKey, attempts, 0)

	if attempts > config.Config.OTP.AUTH.RetryLimit {
		log.Printf("Retry limit exceeded for UUID: %s", uuid)
	}
}

func (h *PublicHandler) RefreshToken(c *gin.Context) {
	auth.RefreshTokenHandler(c)
}
