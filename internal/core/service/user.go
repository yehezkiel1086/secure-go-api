package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

var _ port.UserService = (*UserService)(nil)

// UserService coordinates user domain operations.
type UserService struct {
	userRepo port.UserRepository
}

// NewUserService creates a new UserService with the injected UserRepository port.
func NewUserService(userRepo port.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// RegisterUser registers a new user with bcrypt-hashed password and default User role.
func (s *UserService) RegisterUser(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user already exists
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("checking existing email: %w", err)
	}

	// Hash password using core bcrypt utility
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &domain.User{
		Name:         strings.TrimSpace(req.Name),
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         domain.RoleUser,
	}

	created, err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return created.ToResponse(), nil
}

// GetUserByID retrieves a user by their UUID primary key.
func (s *UserService) GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.UserResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user.ToResponse(), nil
}

// GetUsers returns a paginated list of users along with pagination metadata.
func (s *UserService) GetUsers(ctx context.Context, page, pageSize int32) (*domain.PaginatedUsersResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	limit := pageSize

	users, err := s.userRepo.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	total, err := s.userRepo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting users: %w", err)
	}

	totalPages := int32(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	resUsers := make([]*domain.UserResponse, len(users))
	for i, u := range users {
		resUsers[i] = u.ToResponse()
	}

	return &domain.PaginatedUsersResponse{
		Users:      resUsers,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateUserName updates a user's display name.
func (s *UserService) UpdateUserName(ctx context.Context, id pgtype.UUID, req *domain.UpdateUserNameRequest) (*domain.UserResponse, error) {
	name := strings.TrimSpace(req.Name)
	user, err := s.userRepo.UpdateUserName(ctx, id, name)
	if err != nil {
		return nil, err
	}
	return user.ToResponse(), nil
}

// VerifyEmail verifies a user's email matching a SHA-256 hashed verification token.
func (s *UserService) VerifyEmail(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.ErrInvalidToken
	}

	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	user, err := s.userRepo.GetUserByEmailVerifyToken(ctx, pgtype.Text{String: tokenHash, Valid: true})
	if err != nil {
		return err
	}

	if user.EmailVerifyTokenExpiresAt.Valid && time.Now().After(user.EmailVerifyTokenExpiresAt.Time) {
		return domain.ErrTokenExpired
	}

	if err := s.userRepo.MarkEmailVerified(ctx, user.ID); err != nil {
		return fmt.Errorf("marking email verified: %w", err)
	}

	return nil
}
