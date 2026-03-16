package strutils

import "strings"

func ExtractNameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}

func IsEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}