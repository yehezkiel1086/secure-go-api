package port

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

type AuthRepository interface {
	CreateRefreshToken(ctx context.Context, userID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*domain.RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id pgtype.UUID) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID pgtype.UUID) error
	DeleteExpiredRefreshTokens(ctx context.Context) error
	CountActiveRefreshTokens(ctx context.Context, userID pgtype.UUID) (int64, error)
}

type AuthService interface {
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)
	RefreshToken(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, rawRefreshToken string) error
}
