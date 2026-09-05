package domain

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type RefreshToken struct {
	ID        pgtype.UUID
	UserID    pgtype.UUID
	TokenHash string
	IsRevoked bool
	ExpiresAt pgtype.Timestamptz
	CreatedAt pgtype.Timestamptz
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenPair struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

type LoginResponse struct {
	User   *UserResponse `json:"user"`
	Tokens *TokenPair    `json:"tokens"`
}
