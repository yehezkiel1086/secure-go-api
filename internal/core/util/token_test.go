package util_test

import (
	"testing"

	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

func TestGenerateSecureToken(t *testing.T) {
	// Default 32 bytes should produce a 64-char hex string
	token1, err := util.GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(token1) != 64 {
		t.Fatalf("expected token length 64, got %d", len(token1))
	}

	// Two consecutive tokens must be unique (cryptographic entropy)
	token2, err := util.GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected distinct cryptographic random tokens, got collision")
	}

	// Custom length (e.g. 16 bytes = 32 hex chars)
	token3, err := util.GenerateSecureToken(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token3) != 32 {
		t.Fatalf("expected token length 32, got %d", len(token3))
	}
}

func TestHashToken(t *testing.T) {
	rawToken := "sampletoken12345"
	hash1 := util.HashToken(rawToken)
	hash2 := util.HashToken(rawToken)

	if len(hash1) != 64 {
		t.Fatalf("expected 64-char hex SHA-256 hash, got %d", len(hash1))
	}

	if hash1 != hash2 {
		t.Fatal("expected deterministic SHA-256 hash output")
	}

	if hash1 == rawToken {
		t.Fatal("hash should not equal raw token")
	}
}
