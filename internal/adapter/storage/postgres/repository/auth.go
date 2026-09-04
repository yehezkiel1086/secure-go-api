package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres/sqlc"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

var _ port.AuthRepository = (*AuthRepository)(nil)

// AuthRepository implements port.AuthRepository using sqlc-generated queries.
type AuthRepository struct {
	queries *sqlc.Queries
}

// NewAuthRepository creates an AuthRepository using sqlc Queries.
func NewAuthRepository(queries *sqlc.Queries) *AuthRepository {
	return &AuthRepository{queries: queries}
}

// NewAuthRepositoryWithDB creates an AuthRepository using a DBTX connection.
func NewAuthRepositoryWithDB(db sqlc.DBTX) *AuthRepository {
	return &AuthRepository{queries: sqlc.New(db)}
}

// CreateRefreshToken persists a new hashed refresh token record.
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, userID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*domain.RefreshToken, error) {
	row, err := r.queries.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	return &domain.RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		IsRevoked: row.IsRevoked,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

// GetRefreshToken retrieves a refresh token row by its SHA-256 hash.
func (r *AuthRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	row, err := r.queries.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	return &domain.RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		IsRevoked: row.IsRevoked,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

// RevokeRefreshToken invalidates a single refresh token by ID.
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, id pgtype.UUID) error {
	if err := r.queries.RevokeRefreshToken(ctx, id); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllUserRefreshTokens invalidates all active sessions for the user.
func (r *AuthRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID pgtype.UUID) error {
	if err := r.queries.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return fmt.Errorf("revoke all user refresh tokens: %w", err)
	}
	return nil
}

// DeleteExpiredRefreshTokens purges expired and revoked tokens from the table.
func (r *AuthRepository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	if err := r.queries.DeleteExpiredRefreshTokens(ctx); err != nil {
		return fmt.Errorf("delete expired refresh tokens: %w", err)
	}
	return nil
}

// CountActiveRefreshTokens returns the number of active, non-revoked sessions for a user.
func (r *AuthRepository) CountActiveRefreshTokens(ctx context.Context, userID pgtype.UUID) (int64, error) {
	count, err := r.queries.CountActiveRefreshTokens(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count active refresh tokens: %w", err)
	}
	return count, nil
}
