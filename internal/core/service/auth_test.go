package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/service"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

var testAuthJWTConfig = &config.JWT{
	AccessTokenSecret:    "supersecretaccesstokenkey1234567890",
	RefreshTokenSecret:   "supersecretrefreshtokenkey1234567890",
	AccessTokenDuration:  "15",
	RefreshTokenDuration: "7",
}

type mockAuthRepo struct {
	createRefreshTokenFn         func(ctx context.Context, userID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*domain.RefreshToken, error)
	getRefreshTokenFn            func(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	revokeRefreshTokenFn         func(ctx context.Context, id pgtype.UUID) error
	revokeAllUserRefreshTokensFn func(ctx context.Context, userID pgtype.UUID) error
	deleteExpiredRefreshTokensFn func(ctx context.Context) error
	countActiveRefreshTokensFn   func(ctx context.Context, userID pgtype.UUID) (int64, error)
}

var _ port.AuthRepository = (*mockAuthRepo)(nil)

func (m *mockAuthRepo) CreateRefreshToken(ctx context.Context, userID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*domain.RefreshToken, error) {
	if m.createRefreshTokenFn != nil {
		return m.createRefreshTokenFn(ctx, userID, tokenHash, expiresAt)
	}
	return &domain.RefreshToken{
		ID:        pgtype.UUID{Bytes: [16]byte{10}, Valid: true},
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}, nil
}

func (m *mockAuthRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	if m.getRefreshTokenFn != nil {
		return m.getRefreshTokenFn(ctx, tokenHash)
	}
	return nil, domain.ErrNotFound
}

func (m *mockAuthRepo) RevokeRefreshToken(ctx context.Context, id pgtype.UUID) error {
	if m.revokeRefreshTokenFn != nil {
		return m.revokeRefreshTokenFn(ctx, id)
	}
	return nil
}

func (m *mockAuthRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID pgtype.UUID) error {
	if m.revokeAllUserRefreshTokensFn != nil {
		return m.revokeAllUserRefreshTokensFn(ctx, userID)
	}
	return nil
}

func (m *mockAuthRepo) DeleteExpiredRefreshTokens(ctx context.Context) error {
	if m.deleteExpiredRefreshTokensFn != nil {
		return m.deleteExpiredRefreshTokensFn(ctx)
	}
	return nil
}

func (m *mockAuthRepo) CountActiveRefreshTokens(ctx context.Context, userID pgtype.UUID) (int64, error) {
	if m.countActiveRefreshTokensFn != nil {
		return m.countActiveRefreshTokensFn(ctx, userID)
	}
	return 0, nil
}

func TestAuthService_Login_Success(t *testing.T) {
	passwordHash, _ := util.HashPassword("secretpassword")
	userID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}

	userRepo := &mockUserRepo{
		getUserByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:              userID,
				Name:            "John Doe",
				Email:           email,
				PasswordHash:    passwordHash,
				Role:            domain.RoleUser,
				IsEmailVerified: true,
			}, nil
		},
	}

	persistedTokenHash := ""
	authRepo := &mockAuthRepo{
		createRefreshTokenFn: func(ctx context.Context, uID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*domain.RefreshToken, error) {
			persistedTokenHash = tokenHash
			return &domain.RefreshToken{
				UserID:    uID,
				TokenHash: tokenHash,
				ExpiresAt: expiresAt,
			}, nil
		},
	}

	authService := service.NewAuthService(authRepo, userRepo, testAuthJWTConfig)
	res, err := authService.Login(context.Background(), &domain.LoginRequest{
		Email:    "john@example.com",
		Password: "secretpassword",
	})
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}

	if res.Tokens.AccessToken == "" || res.Tokens.RefreshToken == "" {
		t.Fatal("expected both tokens to be populated")
	}
	if persistedTokenHash == "" {
		t.Fatal("expected refresh token hash to be persisted in database")
	}
	if persistedTokenHash == res.Tokens.RefreshToken {
		t.Fatal("raw refresh token must NOT be stored directly in the database")
	}
}

func TestAuthService_Login_UnverifiedEmail(t *testing.T) {
	passwordHash, _ := util.HashPassword("secretpassword")
	userID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}

	userRepo := &mockUserRepo{
		getUserByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:              userID,
				Name:            "Unverified User",
				Email:           email,
				PasswordHash:    passwordHash,
				Role:            domain.RoleUser,
				IsEmailVerified: false, // Unverified
			}, nil
		},
	}
	authRepo := &mockAuthRepo{}

	authService := service.NewAuthService(authRepo, userRepo, testAuthJWTConfig)
	_, err := authService.Login(context.Background(), &domain.LoginRequest{
		Email:    "unverified@example.com",
		Password: "secretpassword",
	})

	if !errors.Is(err, domain.ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	passwordHash, _ := util.HashPassword("secretpassword")

	userRepo := &mockUserRepo{
		getUserByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				PasswordHash: passwordHash,
			}, nil
		},
	}
	authRepo := &mockAuthRepo{}

	authService := service.NewAuthService(authRepo, userRepo, testAuthJWTConfig)
	_, err := authService.Login(context.Background(), &domain.LoginRequest{
		Email:    "john@example.com",
		Password: "wrongpassword",
	})

	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_RefreshToken_NormalRotation(t *testing.T) {
	userID := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	user := &domain.User{
		ID:              userID,
		Name:            "Jane Doe",
		Email:           "jane@example.com",
		Role:            domain.RoleUser,
		IsEmailVerified: true,
	}

	userRepo := &mockUserRepo{
		getUserByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.User, error) {
			return user, nil
		},
	}

	rawRefreshToken, _, _ := util.GenerateTokenWithExpiry(testAuthJWTConfig, util.TokenRefresh, user)
	tokenHash := util.HashToken(rawRefreshToken)

	revokedTokenID := pgtype.UUID{}
	newCreatedTokenHash := ""

	authRepo := &mockAuthRepo{
		getRefreshTokenFn: func(ctx context.Context, h string) (*domain.RefreshToken, error) {
			if h == tokenHash {
				return &domain.RefreshToken{
					ID:        pgtype.UUID{Bytes: [16]byte{20}, Valid: true},
					UserID:    userID,
					TokenHash: tokenHash,
					IsRevoked: false,
					ExpiresAt: pgtype.Timestamptz{
						Time:  time.Now().Add(24 * time.Hour),
						Valid: true,
					},
				}, nil
			}
			return nil, domain.ErrNotFound
		},
		revokeRefreshTokenFn: func(ctx context.Context, id pgtype.UUID) error {
			revokedTokenID = id
			return nil
		},
		createRefreshTokenFn: func(ctx context.Context, uID pgtype.UUID, h string, exp pgtype.Timestamptz) (*domain.RefreshToken, error) {
			newCreatedTokenHash = h
			return &domain.RefreshToken{UserID: uID, TokenHash: h}, nil
		},
	}

	authService := service.NewAuthService(authRepo, userRepo, testAuthJWTConfig)
	newTokens, err := authService.RefreshToken(context.Background(), rawRefreshToken)
	if err != nil {
		t.Fatalf("unexpected error during refresh: %v", err)
	}

	if revokedTokenID.Bytes != [16]byte{20} {
		t.Errorf("expected consumed token to be revoked, got %v", revokedTokenID)
	}
	if newCreatedTokenHash == "" {
		t.Fatal("expected new refresh token to be persisted")
	}
	if newTokens.RefreshToken == rawRefreshToken {
		t.Fatal("expected rotated refresh token to be different from original")
	}
}

func TestAuthService_RefreshToken_ReuseDetection(t *testing.T) {
	userID := pgtype.UUID{Bytes: [16]byte{5}, Valid: true}
	user := &domain.User{
		ID:    userID,
		Name:  "Compromised User",
		Email: "comp@example.com",
		Role:  domain.RoleUser,
	}

	rawRefreshToken, _, _ := util.GenerateTokenWithExpiry(testAuthJWTConfig, util.TokenRefresh, user)
	tokenHash := util.HashToken(rawRefreshToken)

	allSessionsWiped := false

	authRepo := &mockAuthRepo{
		getRefreshTokenFn: func(ctx context.Context, h string) (*domain.RefreshToken, error) {
			return &domain.RefreshToken{
				ID:        pgtype.UUID{Bytes: [16]byte{55}, Valid: true},
				UserID:    userID,
				TokenHash: tokenHash,
				IsRevoked: true, // ALREADY REVOKED TOKEN REPLAYED!
				ExpiresAt: pgtype.Timestamptz{
					Time:  time.Now().Add(24 * time.Hour),
					Valid: true,
				},
			}, nil
		},
		revokeAllUserRefreshTokensFn: func(ctx context.Context, uID pgtype.UUID) error {
			allSessionsWiped = true
			return nil
		},
	}
	userRepo := &mockUserRepo{}

	authService := service.NewAuthService(authRepo, userRepo, testAuthJWTConfig)
	_, err := authService.RefreshToken(context.Background(), rawRefreshToken)

	if !errors.Is(err, domain.ErrTokenReuse) {
		t.Fatalf("expected ErrTokenReuse, got %v", err)
	}
	if !allSessionsWiped {
		t.Fatal("expected all sessions for compromised user to be wiped upon token reuse detection")
	}
}

func TestAuthService_Logout(t *testing.T) {
	userID := pgtype.UUID{Bytes: [16]byte{9}, Valid: true}
	rawToken := "sometokentologout"
	tokenHash := util.HashToken(rawToken)

	sessionsRevoked := false

	authRepo := &mockAuthRepo{
		getRefreshTokenFn: func(ctx context.Context, h string) (*domain.RefreshToken, error) {
			return &domain.RefreshToken{
				UserID:    userID,
				TokenHash: tokenHash,
			}, nil
		},
		revokeAllUserRefreshTokensFn: func(ctx context.Context, uID pgtype.UUID) error {
			sessionsRevoked = true
			return nil
		},
	}
	userRepo := &mockUserRepo{}

	authService := service.NewAuthService(authRepo, userRepo, testAuthJWTConfig)
	err := authService.Logout(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("unexpected logout error: %v", err)
	}
	if !sessionsRevoked {
		t.Fatal("expected user sessions to be revoked on logout")
	}
}
