package domain

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Supports token rotation + reuse detection. Reused tokens trigger full session revocation.
type RefreshToken struct {
	ID        pgtype.UUID
	UserID    pgtype.UUID
	TokenHash string // SHA-256 hash — raw token never persisted
	IsRevoked bool   // Set on reuse detection (ErrTokenReuse)
	ExpiresAt pgtype.Timestamptz
	CreatedAt pgtype.Timestamptz
}

// LoginRequest defines input for user authentication.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest defines optional JSON payload for token rotation if not using cookies.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TokenPair contains the issued access and refresh tokens along with expiry timestamps.
type TokenPair struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// LoginResponse contains authenticated user details and token pair.
type LoginResponse struct {
	User   *UserResponse `json:"user"`
	Tokens *TokenPair    `json:"tokens"`
}
