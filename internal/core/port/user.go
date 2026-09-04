package port

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

// UserRepository is the driven port interface that defines database operations for users.
type UserRepository interface {
	// CreateUser inserts a new user record and returns the created user entity.
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)

	// GetUserByID retrieves a user by primary key ID.
	GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.User, error)

	// GetUserByEmail retrieves a user by their unique email address.
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetUserByEmailVerifyToken retrieves a user matching the provided email verification token hash.
	GetUserByEmailVerifyToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error)

	// GetUserByPasswordResetToken retrieves a user matching the provided password reset token hash.
	GetUserByPasswordResetToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error)

	// ListUsers returns a paginated list of users ordered by created_at DESC.
	ListUsers(ctx context.Context, limit, offset int32) ([]*domain.User, error)

	// CountUsers returns the total count of users in the system.
	CountUsers(ctx context.Context) (int64, error)

	// UpdateUserName updates a user's display name and returns the updated user entity.
	UpdateUserName(ctx context.Context, id pgtype.UUID, name string) (*domain.User, error)

	// UpdateUserPassword updates a user's password hash and invalidates any reset token.
	UpdateUserPassword(ctx context.Context, id pgtype.UUID, passwordHash string) error

	// SetEmailVerifyToken sets the email verification token hash and expiry for a user.
	SetEmailVerifyToken(ctx context.Context, id pgtype.UUID, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error

	// MarkEmailVerified marks a user's email as verified and clears verification tokens.
	MarkEmailVerified(ctx context.Context, id pgtype.UUID) error

	// SetPasswordResetToken sets the password reset token hash and expiry by user email.
	SetPasswordResetToken(ctx context.Context, email string, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error
}

// UserService is the driving port interface for user domain logic.
type UserService interface {
	// RegisterUser handles new user registration, hashes password, and creates the record.
	RegisterUser(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error)

	// GetUserByID retrieves user profile details by UUID.
	GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.UserResponse, error)

	// GetUsers returns a paginated list of users.
	GetUsers(ctx context.Context, page, pageSize int32) (*domain.PaginatedUsersResponse, error)

	// UpdateUserName updates a user's display name.
	UpdateUserName(ctx context.Context, id pgtype.UUID, req *domain.UpdateUserNameRequest) (*domain.UserResponse, error)

	// VerifyEmail confirms a user's email using a raw verification token.
	VerifyEmail(ctx context.Context, token string) error
}
