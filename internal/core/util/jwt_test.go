package util_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

var testJWTConfig = &config.JWT{
	AccessTokenSecret:    "supersecretaccesstokenkey1234567890",
	RefreshTokenSecret:   "supersecretrefreshtokenkey1234567890",
	AccessTokenDuration:  "15",
	RefreshTokenDuration: "7",
}

func TestJWT_GenerateAndParseAccessToken(t *testing.T) {
	user := &domain.User{
		ID:    pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
		Name:  "Test User",
		Email: "test@example.com",
		Role:  domain.RoleUser,
	}

	token, err := util.GenerateToken(testJWTConfig, util.TokenAccess, user)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := util.ParseToken(testJWTConfig, util.TokenAccess, token)
	if err != nil {
		t.Fatalf("unexpected error parsing token: %v", err)
	}

	if claims.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, claims.Email)
	}
	if claims.Username != user.Name {
		t.Errorf("expected name %s, got %s", user.Name, claims.Username)
	}
	if claims.Role != domain.RoleUser {
		t.Errorf("expected role %d, got %d", domain.RoleUser, claims.Role)
	}
	if claims.Subject != "01020304-0506-0708-090a-0b0c0d0e0f10" {
		t.Errorf("expected subject UUID, got %s", claims.Subject)
	}
}

func TestJWT_GenerateTokenPair(t *testing.T) {
	user := &domain.User{
		ID:    pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		Name:  "Admin User",
		Email: "admin@example.com",
		Role:  domain.RoleAdmin,
	}

	pair, err := util.GenerateTokenPair(testJWTConfig, user)
	if err != nil {
		t.Fatalf("unexpected error generating token pair: %v", err)
	}

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens to be non-empty")
	}
	if pair.AccessTokenExpiresAt.Before(time.Now()) {
		t.Fatal("access token expiry should be in the future")
	}
	if pair.RefreshTokenExpiresAt.Before(pair.AccessTokenExpiresAt) {
		t.Fatal("refresh token expiry should be after access token expiry")
	}

	// Verify access token with access secret
	accessClaims, err := util.ParseToken(testJWTConfig, util.TokenAccess, pair.AccessToken)
	if err != nil {
		t.Fatalf("failed to parse access token: %v", err)
	}
	if accessClaims.Role != domain.RoleAdmin {
		t.Errorf("expected admin role, got %d", accessClaims.Role)
	}

	// Verify refresh token with refresh secret
	refreshClaims, err := util.ParseToken(testJWTConfig, util.TokenRefresh, pair.RefreshToken)
	if err != nil {
		t.Fatalf("failed to parse refresh token: %v", err)
	}
	if refreshClaims.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, refreshClaims.Email)
	}
}

func TestJWT_HashToken(t *testing.T) {
	raw := "some-random-token-string"
	hash1 := util.HashToken(raw)
	hash2 := util.HashToken(raw)

	if hash1 == "" {
		t.Fatal("hash should not be empty")
	}
	if hash1 != hash2 {
		t.Fatal("hash should be deterministic")
	}
	if hash1 == raw {
		t.Fatal("hash should not match raw input")
	}
}
