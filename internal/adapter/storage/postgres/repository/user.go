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

var _ port.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	queries *sqlc.Queries
}

func NewUserRepository(queries *sqlc.Queries) *UserRepository {
	return &UserRepository{
		queries: queries,
	}
}

func NewUserRepositoryWithDB(db sqlc.DBTX) *UserRepository {
	return &UserRepository{
		queries: sqlc.New(db),
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &domain.User{
		ID:              row.ID,
		Name:            row.Name,
		Email:           row.Email,
		PasswordHash:    user.PasswordHash,
		Role:            row.Role,
		IsEmailVerified: row.IsEmailVerified,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &domain.User{
		ID:              row.ID,
		Name:            row.Name,
		Email:           row.Email,
		Role:            row.Role,
		IsEmailVerified: row.IsEmailVerified,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &domain.User{
		ID:                          row.ID,
		Name:                        row.Name,
		Email:                       row.Email,
		PasswordHash:                row.PasswordHash,
		Role:                        row.Role,
		IsEmailVerified:             row.IsEmailVerified,
		EmailVerifyTokenHash:        row.EmailVerifyTokenHash,
		EmailVerifyTokenExpiresAt:   row.EmailVerifyTokenExpiresAt,
		PasswordResetTokenHash:      row.PasswordResetTokenHash,
		PasswordResetTokenExpiresAt: row.PasswordResetTokenExpiresAt,
		CreatedAt:                   row.CreatedAt,
		UpdatedAt:                   row.UpdatedAt,
	}, nil
}

func (r *UserRepository) GetUserByEmailVerifyToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error) {
	row, err := r.queries.GetUserByEmailVerifyToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email verify token: %w", err)
	}

	return &domain.User{
		ID:                        row.ID,
		Name:                      row.Name,
		Email:                     row.Email,
		Role:                      row.Role,
		IsEmailVerified:           row.IsEmailVerified,
		EmailVerifyTokenHash:      row.EmailVerifyTokenHash,
		EmailVerifyTokenExpiresAt: row.EmailVerifyTokenExpiresAt,
		UpdatedAt:                 row.UpdatedAt,
	}, nil
}

func (r *UserRepository) GetUserByPasswordResetToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error) {
	row, err := r.queries.GetUserByPasswordResetToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get user by password reset token: %w", err)
	}

	return &domain.User{
		ID:                          row.ID,
		Email:                       row.Email,
		PasswordResetTokenHash:      row.PasswordResetTokenHash,
		PasswordResetTokenExpiresAt: row.PasswordResetTokenExpiresAt,
	}, nil
}

func (r *UserRepository) ListUsers(ctx context.Context, limit, offset int32) ([]*domain.User, error) {
	rows, err := r.queries.ListUsers(ctx, sqlc.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]*domain.User, len(rows))
	for i, row := range rows {
		users[i] = &domain.User{
			ID:              row.ID,
			Name:            row.Name,
			Email:           row.Email,
			Role:            row.Role,
			IsEmailVerified: row.IsEmailVerified,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}
	}

	return users, nil
}

func (r *UserRepository) CountUsers(ctx context.Context) (int64, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *UserRepository) UpdateUserName(ctx context.Context, id pgtype.UUID, name string) (*domain.User, error) {
	row, err := r.queries.UpdateUserName(ctx, sqlc.UpdateUserNameParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update user name: %w", err)
	}

	return &domain.User{
		ID:              row.ID,
		Name:            row.Name,
		Email:           row.Email,
		Role:            row.Role,
		IsEmailVerified: row.IsEmailVerified,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *UserRepository) UpdateUserPassword(ctx context.Context, id pgtype.UUID, passwordHash string) error {
	err := r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	return nil
}

func (r *UserRepository) SetEmailVerifyToken(ctx context.Context, id pgtype.UUID, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error {
	err := r.queries.SetEmailVerifyToken(ctx, sqlc.SetEmailVerifyTokenParams{
		ID:                        id,
		EmailVerifyTokenHash:      tokenHash,
		EmailVerifyTokenExpiresAt: expiresAt,
	})
	if err != nil {
		return fmt.Errorf("set email verify token: %w", err)
	}
	return nil
}

func (r *UserRepository) MarkEmailVerified(ctx context.Context, id pgtype.UUID) error {
	err := r.queries.MarkEmailVerified(ctx, id)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	return nil
}

func (r *UserRepository) SetPasswordResetToken(ctx context.Context, email string, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error {
	err := r.queries.SetPasswordResetToken(ctx, sqlc.SetPasswordResetTokenParams{
		Email:                       email,
		PasswordResetTokenHash:      tokenHash,
		PasswordResetTokenExpiresAt: expiresAt,
	})
	if err != nil {
		return fmt.Errorf("set password reset token: %w", err)
	}
	return nil
}
