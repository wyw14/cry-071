package service

import (
	"regexp"
	"strings"
)

var (
	phonePattern = regexp.MustCompile(`(?i)(?:\+?86[- ]?)?1[3-9][0-9][- ]?[0-9]{4}[- ]?[0-9]{4}`)
	emailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	idPattern    = regexp.MustCompile(`\b[0-9]{6}(?:19|20)[0-9]{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12][0-9]|3[01])[0-9]{3}[0-9Xx]\b`)
)

type Redactor struct{}

func (Redactor) Text(value string) string {
	value = phonePattern.ReplaceAllStringFunc(value, maskPhone)
	value = emailPattern.ReplaceAllStringFunc(value, maskEmail)
	value = idPattern.ReplaceAllString(value, "******************")
	return value
}

func (Redactor) Contact(value string) string {
	value = strings.TrimSpace(value)
	if phonePattern.MatchString(value) {
		return maskPhone(value)
	}
	if emailPattern.MatchString(value) {
		return maskEmail(value)
	}
	if len([]rune(value)) <= 2 {
		return "**"
	}
	runes := []rune(value)
	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}

func maskPhone(value string) string {
	digits := make([]rune, 0, len(value))
	for _, current := range value {
		if current >= '0' && current <= '9' {
			digits = append(digits, current)
		}
	}
	if len(digits) < 7 {
		return "***"
	}
	return string(digits[:3]) + "****" + string(digits[len(digits)-4:])
}

func maskEmail(value string) string {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	local := []rune(parts[0])
	if len(local) == 0 {
		return "***@" + parts[1]
	}
	return string(local[0]) + "***@" + parts[1]
}
