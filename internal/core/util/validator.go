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

// ValidateEmail performs strict validation on email addresses according to RFC specifications:
// 1. Length constraint: total length must not exceed 254 characters (RFC 5321).
// 2. Local-part length constraint: must not exceed 64 characters.
// 3. Header injection protection: prevents CRLF, tabs, or display-name smuggling.
// 4. Strict domain structure: requires valid domain label, no consecutive dots, and valid TLD.
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return domain.ErrInvalidEmailFormat
	}

	// 1. RFC 5321 length limit
	if len(email) > 254 {
		return domain.ErrInvalidEmailFormat
	}

	// 2. Prevent CRLF / header injection
	if strings.ContainsAny(email, "\r\n\t") {
		return domain.ErrInvalidEmailFormat
	}

	// 3. RFC 5322 address parse and exact match check
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return domain.ErrInvalidEmailFormat
	}

	// 4. Local part and domain split
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return domain.ErrInvalidEmailFormat
	}

	localPart := parts[0]
	domainPart := parts[1]

	// 5. Local part length limit (RFC 5321: max 64 octets)
	if len(localPart) == 0 || len(localPart) > 64 {
		return domain.ErrInvalidEmailFormat
	}

	// 6. Domain structure checks
	if len(domainPart) == 0 || len(domainPart) > 253 {
		return domain.ErrInvalidEmailFormat
	}

	if strings.Contains(domainPart, "..") {
		return domain.ErrInvalidEmailFormat
	}

	if !tldRegex.MatchString(domainPart) {
		return domain.ErrInvalidEmailFormat
	}

	// 7. Overall regex pattern check
	if !emailRegex.MatchString(email) {
		return domain.ErrInvalidEmailFormat
	}

	return nil
}
