package port

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

// AuthRepository is the driven port interface that defines database operations for refresh tokens.
type AuthRepository interface {
	// CreateRefreshToken inserts a new hashed refresh token record.
	CreateRefreshToken(ctx context.Context, userID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*domain.RefreshToken, error)

	// GetRefreshToken locates a refresh token record by its SHA-256 hash.
	GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)

	// RevokeRefreshToken sets is_revoked = TRUE for a single token row.
	RevokeRefreshToken(ctx context.Context, id pgtype.UUID) error

	// RevokeAllUserRefreshTokens invalidates all active sessions for a user (on logout or ErrTokenReuse).
	RevokeAllUserRefreshTokens(ctx context.Context, userID pgtype.UUID) error

	// DeleteExpiredRefreshTokens maintenance query to purge expired revoked tokens.
	DeleteExpiredRefreshTokens(ctx context.Context) error

	// CountActiveRefreshTokens counts unrevoked unexpired tokens for a user.
	CountActiveRefreshTokens(ctx context.Context, userID pgtype.UUID) (int64, error)
}

// AuthService is the driving port interface for authentication workflows.
type AuthService interface {
	// Login validates credentials and issues an access/refresh token pair.
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)

	// RefreshToken validates a refresh token, revokes it, detects reuse, and issues a fresh token pair.
	RefreshToken(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error)

	// Logout invalidates all user refresh tokens for the session and clears cookies.
	Logout(ctx context.Context, rawRefreshToken string) error
}
