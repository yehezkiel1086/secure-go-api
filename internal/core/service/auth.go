package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

var _ port.AuthService = (*AuthService)(nil)

type AuthService struct {
	authRepo port.AuthRepository
	userRepo port.UserRepository
	jwtCfg   *config.JWT
}

func NewAuthService(authRepo port.AuthRepository, userRepo port.UserRepository, jwtCfg *config.JWT) *AuthService {
	return &AuthService{
		authRepo: authRepo,
		userRepo: userRepo,
		jwtCfg:   jwtCfg,
	}
}

func (s *AuthService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("getting user by email: %w", err)
	}

	if err := util.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsEmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	tokens, err := util.GenerateTokenPair(s.jwtCfg, user)
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	tokenHash := util.HashToken(tokens.RefreshToken)
	expiresAt := pgtype.Timestamptz{
		Time:  tokens.RefreshTokenExpiresAt,
		Valid: true,
	}

	if _, err := s.authRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("persisting refresh token: %w", err)
	}

	return &domain.LoginResponse{
		User:   user.ToResponse(),
		Tokens: tokens,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error) {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return nil, domain.ErrInvalidToken
	}

	_, err := util.ParseToken(s.jwtCfg, util.TokenRefresh, rawRefreshToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	tokenHash := util.HashToken(rawRefreshToken)
	tokenRecord, err := s.authRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("locating refresh token: %w", err)
	}

	// token reuse detection: revoke all sessions if a revoked token is presented
	if tokenRecord.IsRevoked {
		_ = s.authRepo.RevokeAllUserRefreshTokens(ctx, tokenRecord.UserID)
		return nil, domain.ErrTokenReuse
	}

	if tokenRecord.ExpiresAt.Valid && time.Now().After(tokenRecord.ExpiresAt.Time) {
		return nil, domain.ErrTokenExpired
	}

	if err := s.authRepo.RevokeRefreshToken(ctx, tokenRecord.ID); err != nil {
		return nil, fmt.Errorf("revoking current token: %w", err)
	}

	user, err := s.userRepo.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	if !user.IsEmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	newTokens, err := util.GenerateTokenPair(s.jwtCfg, user)
	if err != nil {
		return nil, fmt.Errorf("generating new tokens: %w", err)
	}

	newTokenHash := util.HashToken(newTokens.RefreshToken)
	newExpiresAt := pgtype.Timestamptz{
		Time:  newTokens.RefreshTokenExpiresAt,
		Valid: true,
	}

	if _, err := s.authRepo.CreateRefreshToken(ctx, user.ID, newTokenHash, newExpiresAt); err != nil {
		return nil, fmt.Errorf("persisting new refresh token: %w", err)
	}

	return newTokens, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return nil
	}

	tokenHash := util.HashToken(rawRefreshToken)
	tokenRecord, err := s.authRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("looking up token for logout: %w", err)
	}

	if err := s.authRepo.RevokeAllUserRefreshTokens(ctx, tokenRecord.UserID); err != nil {
		return fmt.Errorf("revoking user tokens on logout: %w", err)
	}

	return nil
}
