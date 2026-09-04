package domain

import "github.com/jackc/pgx/v5/pgtype"

// Supports token rotation + reuse detection. Reused tokens trigger full session revocation.
type RefreshToken struct {
	ID     pgtype.UUID
	UserID pgtype.UUID
	// SHA-256 hash — raw token never persisted
	TokenHash string
	// Set on reuse detection (ErrTokenReuse)
	IsRevoked bool
	ExpiresAt pgtype.Timestamptz
	CreatedAt pgtype.Timestamptz
}
