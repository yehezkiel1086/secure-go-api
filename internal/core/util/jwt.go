package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

type Token string

const (
	TokenAccess  Token = "access_token"
	TokenRefresh Token = "refresh_token"
)

type JWTClaims struct {
	Username string `json:"username"`
	Role     int32  `json:"role"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// HashToken generates a SHA-256 hex string of the given raw token.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GenerateToken generates a signed JWT string for the given token type.
func GenerateToken(cfg *config.JWT, tokenType Token, user *domain.User) (string, error) {
	tokenStr, _, err := GenerateTokenWithExpiry(cfg, tokenType, user)
	return tokenStr, err
}

// GenerateTokenWithExpiry generates a signed JWT string and returns its exact expiration time.
func GenerateTokenWithExpiry(cfg *config.JWT, tokenType Token, user *domain.User) (string, time.Time, error) {
	var signingKey []byte
	var duration time.Duration

	switch tokenType {
	case TokenAccess:
		signingKey = []byte(cfg.AccessTokenSecret)
		d, err := strconv.Atoi(cfg.AccessTokenDuration)
		if err != nil || d <= 0 {
			d = 15 // default 15 minutes
		}
		duration = time.Duration(d) * time.Minute
	case TokenRefresh:
		signingKey = []byte(cfg.RefreshTokenSecret)
		d, err := strconv.Atoi(cfg.RefreshTokenDuration)
		if err != nil || d <= 0 {
			d = 7 // default 7 days
		}
		duration = time.Duration(d) * time.Hour * 24
	}

	var subject string
	if user.ID.Valid {
		b := user.ID.Bytes
		subject = fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
			b[0], b[1], b[2], b[3],
			b[4], b[5],
			b[6], b[7],
			b[8], b[9],
			b[10], b[11], b[12], b[13], b[14], b[15],
		)
	}

	now := time.Now()
	expiresAt := now.Add(duration)

	// Cryptographic entropy for unique token ID (jti)
	var jtiBytes [16]byte
	_, _ = rand.Read(jtiBytes[:])
	jti := hex.EncodeToString(jtiBytes[:])

	claims := &JWTClaims{
		Username: user.Name,
		Role:     user.Role,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(signingKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return ss, expiresAt, nil
}

// GenerateTokenPair generates both an access token and a refresh token for the user.
func GenerateTokenPair(cfg *config.JWT, user *domain.User) (*domain.TokenPair, error) {
	accessToken, accessExpiry, err := GenerateTokenWithExpiry(cfg, TokenAccess, user)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, refreshExpiry, err := GenerateTokenWithExpiry(cfg, TokenRefresh, user)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshTokenExpiresAt: refreshExpiry,
	}, nil
}

// ParseToken parses and verifies the token signature and claims for the specified token type.
func ParseToken(cfg *config.JWT, tokenType Token, tokenString string) (*JWTClaims, error) {
	var signingKey []byte

	switch tokenType {
	case TokenAccess:
		signingKey = []byte(cfg.AccessTokenSecret)
	case TokenRefresh:
		signingKey = []byte(cfg.RefreshTokenSecret)
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return signingKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrInvalidKey
	}

	return claims, nil
}
