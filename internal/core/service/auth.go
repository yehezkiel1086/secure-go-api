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

// AuthService coordinates authentication workflows.
type AuthService struct {
	authRepo port.AuthRepository
	userRepo port.UserRepository
	jwtCfg   *config.JWT
}

// NewAuthService creates a new AuthService with the injected repositories and JWT config.
func NewAuthService(authRepo port.AuthRepository, userRepo port.UserRepository, jwtCfg *config.JWT) *AuthService {
	return &AuthService{
		authRepo: authRepo,
		userRepo: userRepo,
		jwtCfg:   jwtCfg,
	}
}

// Login validates user credentials, generates token pair, and persists the refresh token hash.
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

	tokens, err := util.GenerateTokenPair(s.jwtCfg, user)
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	// Persist SHA-256 hash of raw refresh token
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

// RefreshToken validates the refresh token, detects reuse, and issues a rotated token pair.
func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error) {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return nil, domain.ErrInvalidToken
	}

	// 1. Validate JWT signature and structure
	_, err := util.ParseToken(s.jwtCfg, util.TokenRefresh, rawRefreshToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// 2. Hash raw token and lookup record in DB
	tokenHash := util.HashToken(rawRefreshToken)
	tokenRecord, err := s.authRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("locating refresh token: %w", err)
	}

	// 3. Security: Token Reuse Detection
	// If a revoked token is replayed, revoke ALL sessions for this user (potential token theft!)
	if tokenRecord.IsRevoked {
		_ = s.authRepo.RevokeAllUserRefreshTokens(ctx, tokenRecord.UserID)
		return nil, domain.ErrTokenReuse
	}

	// 4. Check expiration
	if tokenRecord.ExpiresAt.Valid && time.Now().After(tokenRecord.ExpiresAt.Time) {
		return nil, domain.ErrTokenExpired
	}

	// 5. Revoke used token during normal rotation
	if err := s.authRepo.RevokeRefreshToken(ctx, tokenRecord.ID); err != nil {
		return nil, fmt.Errorf("revoking current token: %w", err)
	}

	// 6. Look up user to generate fresh pair
	user, err := s.userRepo.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	newTokens, err := util.GenerateTokenPair(s.jwtCfg, user)
	if err != nil {
		return nil, fmt.Errorf("generating new tokens: %w", err)
	}

	// 7. Persist new refresh token hash
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

// Logout invalidates all user refresh tokens for the session.
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
