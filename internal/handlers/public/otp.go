package public

import (
	"fmt"
	"math/rand"
)

// GenerateOTP generates a numeric OTP of the specified length
func GenerateOTP(length int) string {
	if length <= 0 {
		return ""
	}

	// Calculate the maximum value based on the desired length
	maxValue := int64(1)
	for i := 0; i < length; i++ {
		maxValue *= 10
	}

	// Generate random number in the range [0, maxValue) and format it with leading zeros
	randomNumber := rand.Int63n(maxValue)
	return fmt.Sprintf("%0*d", length, randomNumber)
}
