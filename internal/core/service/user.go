package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

const (
	// EmailVerificationTokenDuration defines how long an email verification token remains valid.
	EmailVerificationTokenDuration = 15 * time.Minute
)

var _ port.UserService = (*UserService)(nil)

type UserService struct {
	userRepo       port.UserRepository
	emailPublisher port.EmailPublisher
}

func NewUserService(userRepo port.UserRepository, emailPublisher port.EmailPublisher) *UserService {
	return &UserService{
		userRepo:       userRepo,
		emailPublisher: emailPublisher,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if err := util.ValidateEmail(email); err != nil {
		return nil, domain.ErrInvalidEmailFormat
	}

	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("checking existing email: %w", err)
	}

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

	// generate secure one-time verification token
	rawToken, err := util.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generating verification token: %w", err)
	}

	tokenHash := util.HashToken(rawToken)
	expiresAt := time.Now().Add(EmailVerificationTokenDuration)

	err = s.userRepo.SetEmailVerifyToken(
		ctx,
		created.ID,
		pgtype.Text{String: tokenHash, Valid: true},
		pgtype.Timestamptz{Time: expiresAt, Valid: true},
	)
	if err != nil {
		return nil, fmt.Errorf("saving email verification token: %w", err)
	}

	if s.emailPublisher != nil {
		emailPayload := &domain.EmailPayload{
			To:        created.Email,
			Subject:   "Please verify your email address",
			Body:      fmt.Sprintf("Hello %s,\n\nPlease verify your email address by clicking the link below:\nhttp://localhost:8080/api/v1/confirm-email?token=%s\n\nThis verification link expires in 15 minutes.\n", created.Name, rawToken),
			Token:     rawToken,
			ExpiresAt: expiresAt,
		}

		if err := s.emailPublisher.PublishVerificationEmail(ctx, emailPayload); err != nil {
			slog.Error("failed to enqueue verification email to rabbitmq", "error", err, "email", created.Email)
		}
	}

	return created.ToResponse(), nil
}

func (s *UserService) GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.UserResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user.ToResponse(), nil
}

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

func (s *UserService) UpdateUserName(ctx context.Context, id pgtype.UUID, req *domain.UpdateUserNameRequest) (*domain.UserResponse, error) {
	name := strings.TrimSpace(req.Name)
	user, err := s.userRepo.UpdateUserName(ctx, id, name)
	if err != nil {
		return nil, err
	}
	return user.ToResponse(), nil
}

func (s *UserService) VerifyEmail(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.ErrInvalidToken
	}

	tokenHash := util.HashToken(token)

	user, err := s.userRepo.GetUserByEmailVerifyToken(ctx, pgtype.Text{String: tokenHash, Valid: true})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrInvalidToken
		}
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

func (s *UserService) ResendVerificationEmail(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := util.ValidateEmail(email); err != nil {
		return domain.ErrInvalidEmailFormat
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// do not leak user existence
			return nil
		}
		return fmt.Errorf("retrieving user: %w", err)
	}

	if user.IsEmailVerified {
		return nil
	}

	rawToken, err := util.GenerateSecureToken(32)
	if err != nil {
		return fmt.Errorf("generating verification token: %w", err)
	}

	tokenHash := util.HashToken(rawToken)
	expiresAt := time.Now().Add(EmailVerificationTokenDuration)

	err = s.userRepo.SetEmailVerifyToken(
		ctx,
		user.ID,
		pgtype.Text{String: tokenHash, Valid: true},
		pgtype.Timestamptz{Time: expiresAt, Valid: true},
	)
	if err != nil {
		return fmt.Errorf("updating email verification token: %w", err)
	}

	if s.emailPublisher != nil {
		emailPayload := &domain.EmailPayload{
			To:        user.Email,
			Subject:   "Please verify your email address",
			Body:      fmt.Sprintf("Hello %s,\n\nPlease verify your email address by clicking the link below:\nhttp://localhost:8080/api/v1/confirm-email?token=%s\n\nThis verification link expires in 15 minutes.\n", user.Name, rawToken),
			Token:     rawToken,
			ExpiresAt: expiresAt,
		}

		if err := s.emailPublisher.PublishVerificationEmail(ctx, emailPayload); err != nil {
			slog.Error("failed to enqueue verification email to rabbitmq", "error", err, "email", user.Email)
		}
	}

	return nil
}
