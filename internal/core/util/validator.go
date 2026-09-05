package util

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

// Regex for validating domain part and overall email structure.
var (
	// RFC 5322 compliant regex with strict domain requirements.
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	// TLD regex requiring at least 2 alpha characters.
	tldRegex = regexp.MustCompile(`\.[a-zA-Z]{2,}$`)
)

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return domain.ErrInvalidEmailFormat
	}

	// rfc 5321 length and crlf injection checks
	if len(email) > 254 || strings.ContainsAny(email, "\r\n\t") {
		return domain.ErrInvalidEmailFormat
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return domain.ErrInvalidEmailFormat
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return domain.ErrInvalidEmailFormat
	}

	localPart := parts[0]
	domainPart := parts[1]

	if len(localPart) == 0 || len(localPart) > 64 {
		return domain.ErrInvalidEmailFormat
	}

	if len(domainPart) == 0 || len(domainPart) > 253 || strings.Contains(domainPart, "..") {
		return domain.ErrInvalidEmailFormat
	}

	if !tldRegex.MatchString(domainPart) || !emailRegex.MatchString(email) {
		return domain.ErrInvalidEmailFormat
	}

	return nil
}
