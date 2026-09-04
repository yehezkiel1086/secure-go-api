package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateSecureToken generates a cryptographically random token of byteLen bytes,
// returning it as a hexadecimal string (e.g. 32 bytes yields 64 hex characters, 256 bits entropy).
func GenerateSecureToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = 32 // default 256-bit entropy
	}

	b := make([]byte, byteLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}

	return hex.EncodeToString(b), nil
}
