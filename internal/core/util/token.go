package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateSecureToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = 32
	}

	b := make([]byte, byteLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}

	return hex.EncodeToString(b), nil
}
