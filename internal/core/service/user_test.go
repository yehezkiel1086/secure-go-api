package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/service"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

type mockUserRepo struct {
	createUserFn                func(ctx context.Context, user *domain.User) (*domain.User, error)
	getUserByIDFn               func(ctx context.Context, id pgtype.UUID) (*domain.User, error)
	getUserByEmailFn             func(ctx context.Context, email string) (*domain.User, error)
	getUserByEmailVerifyTokenFn func(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error)
	getUserByPasswordResetTokenFn func(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error)
	listUsersFn                 func(ctx context.Context, limit, offset int32) ([]*domain.User, error)
	countUsersFn                func(ctx context.Context) (int64, error)
	updateUserNameFn            func(ctx context.Context, id pgtype.UUID, name string) (*domain.User, error)
	updateUserPasswordFn        func(ctx context.Context, id pgtype.UUID, passwordHash string) error
	setEmailVerifyTokenFn       func(ctx context.Context, id pgtype.UUID, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error
	markEmailVerifiedFn         func(ctx context.Context, id pgtype.UUID) error
	setPasswordResetTokenFn     func(ctx context.Context, email string, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error
}

var _ port.UserRepository = (*mockUserRepo)(nil)

func (m *mockUserRepo) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, user)
	}
	return user, nil
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetUserByEmailVerifyToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error) {
	if m.getUserByEmailVerifyTokenFn != nil {
		return m.getUserByEmailVerifyTokenFn(ctx, tokenHash)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetUserByPasswordResetToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error) {
	if m.getUserByPasswordResetTokenFn != nil {
		return m.getUserByPasswordResetTokenFn(ctx, tokenHash)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) ListUsers(ctx context.Context, limit, offset int32) ([]*domain.User, error) {
	if m.listUsersFn != nil {
		return m.listUsersFn(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockUserRepo) CountUsers(ctx context.Context) (int64, error) {
	if m.countUsersFn != nil {
		return m.countUsersFn(ctx)
	}
	return 0, nil
}

func (m *mockUserRepo) UpdateUserName(ctx context.Context, id pgtype.UUID, name string) (*domain.User, error) {
	if m.updateUserNameFn != nil {
		return m.updateUserNameFn(ctx, id, name)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) UpdateUserPassword(ctx context.Context, id pgtype.UUID, passwordHash string) error {
	if m.updateUserPasswordFn != nil {
		return m.updateUserPasswordFn(ctx, id, passwordHash)
	}
	return nil
}

func (m *mockUserRepo) SetEmailVerifyToken(ctx context.Context, id pgtype.UUID, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error {
	if m.setEmailVerifyTokenFn != nil {
		return m.setEmailVerifyTokenFn(ctx, id, tokenHash, expiresAt)
	}
	return nil
}

func (m *mockUserRepo) MarkEmailVerified(ctx context.Context, id pgtype.UUID) error {
	if m.markEmailVerifiedFn != nil {
		return m.markEmailVerifiedFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepo) SetPasswordResetToken(ctx context.Context, email string, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error {
	if m.setPasswordResetTokenFn != nil {
		return m.setPasswordResetTokenFn(ctx, email, tokenHash, expiresAt)
	}
	return nil
}

func TestUserService_RegisterUser_Success(t *testing.T) {
	repo := &mockUserRepo{
		getUserByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
		createUserFn: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			// Verify password was hashed and plain text was not saved
			if user.PasswordHash == "password123" {
				t.Fatal("password should be hashed")
			}
			if err := util.ComparePassword(user.PasswordHash, "password123"); err != nil {
				t.Fatalf("hashed password does not match original: %v", err)
			}
			user.ID = pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
			return user, nil
		},
	}

	svc := service.NewUserService(repo)
	res, err := svc.RegisterUser(context.Background(), &domain.RegisterUserRequest{
		Name:     "Alice",
		Email:    "Alice@Example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Name != "Alice" {
		t.Errorf("expected Alice, got %s", res.Name)
	}
	if res.Email != "alice@example.com" {
		t.Errorf("expected lowercase alice@example.com, got %s", res.Email)
	}
	if res.Role != domain.RoleUser {
		t.Errorf("expected role 2001, got %d", res.Role)
	}
}

func TestUserService_RegisterUser_EmailConflict(t *testing.T) {
	repo := &mockUserRepo{
		getUserByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{Email: email}, nil
		},
	}

	svc := service.NewUserService(repo)
	_, err := svc.RegisterUser(context.Background(), &domain.RegisterUserRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
	})

	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestUserService_GetUsers(t *testing.T) {
	repo := &mockUserRepo{
		listUsersFn: func(ctx context.Context, limit, offset int32) ([]*domain.User, error) {
			return []*domain.User{
				{Name: "User 1", Email: "u1@example.com"},
				{Name: "User 2", Email: "u2@example.com"},
			}, nil
		},
		countUsersFn: func(ctx context.Context) (int64, error) {
			return 25, nil
		},
	}

	svc := service.NewUserService(repo)
	res, err := svc.GetUsers(context.Background(), 2, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(res.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(res.Users))
	}
	if res.Total != 25 {
		t.Errorf("expected 25 total, got %d", res.Total)
	}
	if res.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", res.TotalPages)
	}
	if res.Page != 2 {
		t.Errorf("expected page 2, got %d", res.Page)
	}
}

func TestUserService_VerifyEmail(t *testing.T) {
	rawToken := "supersecretverificationtoken"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	markedVerified := false

	repo := &mockUserRepo{
		getUserByEmailVerifyTokenFn: func(ctx context.Context, h pgtype.Text) (*domain.User, error) {
			if h.String != tokenHash {
				t.Fatalf("expected hash %s, got %s", tokenHash, h.String)
			}
			return &domain.User{
				ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
				EmailVerifyTokenExpiresAt: pgtype.Timestamptz{
					Time:  time.Now().Add(10 * time.Minute),
					Valid: true,
				},
			}, nil
		},
		markEmailVerifiedFn: func(ctx context.Context, id pgtype.UUID) error {
			markedVerified = true
			return nil
		},
	}

	svc := service.NewUserService(repo)
	err := svc.VerifyEmail(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !markedVerified {
		t.Fatal("expected user to be marked as verified")
	}
}

func TestUserService_VerifyEmail_Expired(t *testing.T) {
	repo := &mockUserRepo{
		getUserByEmailVerifyTokenFn: func(ctx context.Context, h pgtype.Text) (*domain.User, error) {
			return &domain.User{
				ID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true},
				EmailVerifyTokenExpiresAt: pgtype.Timestamptz{
					Time:  time.Now().Add(-10 * time.Minute), // expired
					Valid: true,
				},
			}, nil
		},
	}

	svc := service.NewUserService(repo)
	err := svc.VerifyEmail(context.Background(), "sometoken")
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}
