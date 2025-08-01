package otp

import (
	"regexp"
)

// Find all 6-digit OTPs in the input string
func Extract(text string) string {
	re := regexp.MustCompile(`\b\d{6}\b`)

	return re.FindString(text)
}
