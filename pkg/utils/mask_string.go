package utils

import "strings"

// Masked string generator
func maskString(visibleStart, visibleEnd int, str string) string {
	if len(str) <= visibleStart+visibleEnd {
		return str
	}
	maskedPart := strings.Repeat("*", 5)
	return str[:visibleStart] + maskedPart + str[len(str)-visibleEnd:]
}
