package public

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"encrypted-db/config"
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
	ID     int
	Status string
	Info   map[string]interface{}
}

// GenerateRedisKey generates a Redis key by hashing operation, contact, and OTP
func GenerateRedisKey(operation, contact, otp string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", operation, contact, otp)))
	return hex.EncodeToString(hash[:])
}

// // Masked string generator
// func maskString(visibleStart, visibleEnd int, str string) string {
// 	if len(str) <= visibleStart+visibleEnd {
// 		return str
// 	}
// 	maskedPart := strings.Repeat("*", 5)
// 	return str[:visibleStart] + maskedPart + str[len(str)-visibleEnd:]
// }

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

	redisKey := GenerateRedisKey(config.Config.OTP.AUTH.Name, userUUID, otpCode)
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

	redisKey := GenerateRedisKey(config.Config.OTP.AUTH.Name, reqUUID, req.OTP)
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

	h.checkOrCreateUser(c, contactType, contact)
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

// checkOrCreateUser checks if a user exists or creates a new user if not
func (h *PublicHandler) checkOrCreateUser(c *gin.Context, contactType, contact string) {
	log.Println("Checking if user exists or creating a new user")

	query := fmt.Sprintf(`SELECT * FROM users WHERE info->'contact'->>'%s' = $1`, contactType)
	var user User
	err := h.PostgresDB.DB.QueryRow(query, contact).Scan(&user.ID, &user.Status, &user.Info)
	if err == sql.ErrNoRows {
		h.createUser(c, contactType, contact) // Pass `c` to `createUser`
	} else if err != nil {
		log.Printf("Database error: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Database error", nil)
	} else {
		log.Printf("User already exists with ID: %d\n", user.ID)
		helpers.SendResponse(c, http.StatusOK, "Login successful", nil)
	}
}

// createUser inserts a new user in the database
func (h *PublicHandler) createUser(c *gin.Context, contactType, contact string) {
	info := map[string]interface{}{
		"contact": map[string]interface{}{
			contactType: contact,
		},
	}
	infoData, _ := json.Marshal(info)

	query := `INSERT INTO users (info, status) VALUES ($1, 'approved') RETURNING id`
	var userID int
	err := h.PostgresDB.DB.QueryRow(query, infoData).Scan(&userID)
	if err != nil {
		log.Printf("Failed to insert new user: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to create user", nil)
		return
	}

	log.Printf("New user created successfully with ID: %d\n", userID)
	helpers.SendResponse(c, http.StatusCreated, "User registration completed successfully", nil)
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
