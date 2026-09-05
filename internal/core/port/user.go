package port

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
	GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByEmailVerifyToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error)
	GetUserByPasswordResetToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error)
	ListUsers(ctx context.Context, limit, offset int32) ([]*domain.User, error)
	CountUsers(ctx context.Context) (int64, error)
	UpdateUserName(ctx context.Context, id pgtype.UUID, name string) (*domain.User, error)
	UpdateUserPassword(ctx context.Context, id pgtype.UUID, passwordHash string) error
	SetEmailVerifyToken(ctx context.Context, id pgtype.UUID, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error
	MarkEmailVerified(ctx context.Context, id pgtype.UUID) error
	SetPasswordResetToken(ctx context.Context, email string, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error
}

type UserService interface {
	RegisterUser(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error)
	GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.UserResponse, error)
	GetUsers(ctx context.Context, page, pageSize int32) (*domain.PaginatedUsersResponse, error)
	UpdateUserName(ctx context.Context, id pgtype.UUID, req *domain.UpdateUserNameRequest) (*domain.UserResponse, error)
	VerifyEmail(ctx context.Context, token string) error
	ResendVerificationEmail(ctx context.Context, email string) error
}

