package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/handler"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/service"
)

type memoryUserRepo struct {
	usersByEmail map[string]*domain.User
	usersByID    map[string]*domain.User
	tokenToUser  map[string]*domain.User
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{
		usersByEmail: make(map[string]*domain.User),
		usersByID:    make(map[string]*domain.User),
		tokenToUser:  make(map[string]*domain.User),
	}
}

var _ port.UserRepository = (*memoryUserRepo)(nil)

func (r *memoryUserRepo) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	user.ID = pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4}, Valid: true}
	r.usersByEmail[user.Email] = user
	idStr := "01020304-0000-0000-0000-000000000000"
	r.usersByID[idStr] = user
	return user, nil
}

func (r *memoryUserRepo) GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.User, error) {
	for _, u := range r.usersByID {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (r *memoryUserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := r.usersByEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (r *memoryUserRepo) GetUserByEmailVerifyToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error) {
	u, ok := r.tokenToUser[tokenHash.String]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (r *memoryUserRepo) GetUserByPasswordResetToken(ctx context.Context, tokenHash pgtype.Text) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (r *memoryUserRepo) ListUsers(ctx context.Context, limit, offset int32) ([]*domain.User, error) {
	return nil, nil
}

func (r *memoryUserRepo) CountUsers(ctx context.Context) (int64, error) {
	return int64(len(r.usersByEmail)), nil
}

func (r *memoryUserRepo) UpdateUserName(ctx context.Context, id pgtype.UUID, name string) (*domain.User, error) {
	return nil, nil
}

func (r *memoryUserRepo) UpdateUserPassword(ctx context.Context, id pgtype.UUID, passwordHash string) error {
	return nil
}

func (r *memoryUserRepo) SetEmailVerifyToken(ctx context.Context, id pgtype.UUID, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error {
	for _, u := range r.usersByEmail {
		u.EmailVerifyTokenHash = tokenHash
		u.EmailVerifyTokenExpiresAt = expiresAt
		r.tokenToUser[tokenHash.String] = u
	}
	return nil
}

func (r *memoryUserRepo) MarkEmailVerified(ctx context.Context, id pgtype.UUID) error {
	for _, u := range r.usersByEmail {
		u.IsEmailVerified = true
		delete(r.tokenToUser, u.EmailVerifyTokenHash.String)
		u.EmailVerifyTokenHash = pgtype.Text{}
		u.EmailVerifyTokenExpiresAt = pgtype.Timestamptz{}
	}
	return nil
}

func (r *memoryUserRepo) SetPasswordResetToken(ctx context.Context, email string, tokenHash pgtype.Text, expiresAt pgtype.Timestamptz) error {
	return nil
}

type memoryAuthRepo struct {
	tokens map[string]*domain.RefreshToken
}

func newMemoryAuthRepo() *memoryAuthRepo {
	return &memoryAuthRepo{tokens: make(map[string]*domain.RefreshToken)}
}

var _ port.AuthRepository = (*memoryAuthRepo)(nil)

func (r *memoryAuthRepo) CreateRefreshToken(ctx context.Context, userID pgtype.UUID, tokenHash string, expiresAt pgtype.Timestamptz) (*domain.RefreshToken, error) {
	rt := &domain.RefreshToken{
		ID:        pgtype.UUID{Bytes: [16]byte{9, 9}, Valid: true},
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		IsRevoked: false,
	}
	r.tokens[tokenHash] = rt
	return rt, nil
}

func (r *memoryAuthRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	rt, ok := r.tokens[tokenHash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return rt, nil
}

func (r *memoryAuthRepo) RevokeRefreshToken(ctx context.Context, id pgtype.UUID) error {
	return nil
}

func (r *memoryAuthRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID pgtype.UUID) error {
	return nil
}

func (r *memoryAuthRepo) DeleteExpiredRefreshTokens(ctx context.Context) error {
	return nil
}

func (r *memoryAuthRepo) CountActiveRefreshTokens(ctx context.Context, userID pgtype.UUID) (int64, error) {
	return 1, nil
}

type capturingPublisher struct {
	lastPayload *domain.EmailPayload
}

func (p *capturingPublisher) PublishVerificationEmail(ctx context.Context, payload *domain.EmailPayload) error {
	p.lastPayload = payload
	return nil
}

func TestE2E_Registration_EmailVerification_LoginFlow(t *testing.T) {
	conf := &config.Container{
		App: &config.App{Name: "test-app", Env: "test"},
		JWT: testJWTConfig,
	}

	userRepo := newMemoryUserRepo()
	authRepo := newMemoryAuthRepo()
	publisher := &capturingPublisher{}

	userSvc := service.NewUserService(userRepo, publisher)
	authSvc := service.NewAuthService(authRepo, userRepo, conf.JWT)

	userHandler := handler.NewUserHandler(userSvc)
	authHandler := handler.NewAuthHandler(authSvc, conf.JWT, conf.App)

	router := handler.NewRouter(conf, userHandler, authHandler)

	// Step 1: Register with invalid email format -> must return 400 Bad Request
	badReg := domain.RegisterUserRequest{
		Name:     "Charlie",
		Email:    "invalid-email-format",
		Password: "password123",
	}
	badBody, _ := json.Marshal(badReg)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for bad email, got %d", w.Code)
	}

	// Step 2: Register with valid email -> must return 201 Created and publish email with raw token
	validReg := domain.RegisterUserRequest{
		Name:     "Charlie",
		Email:    "charlie@example.com",
		Password: "password123",
	}
	validBody, _ := json.Marshal(validReg)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(validBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	if publisher.lastPayload == nil || publisher.lastPayload.Token == "" {
		t.Fatal("expected verification email with raw token to be published")
	}
	rawToken := publisher.lastPayload.Token

	// Step 3: Attempt to login BEFORE email verification -> must return 403 Forbidden
	loginReq := domain.LoginRequest{
		Email:    "charlie@example.com",
		Password: "password123",
	}
	loginBody, _ := json.Marshal(loginReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when logging in before email verification, got %d: %s", w.Code, w.Body.String())
	}

	// Step 4: Confirm email with invalid token -> must return 400 Bad Request
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/confirm-email?token=invalidrandomtoken", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request with invalid token, got %d", w.Code)
	}

	// Step 5: Confirm email with valid token -> must return 200 OK
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/confirm-email?token="+rawToken, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid token, got %d: %s", w.Code, w.Body.String())
	}

	// Step 6: Login AFTER verification -> must return 200 OK with JWT tokens
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK upon login after verification, got %d: %s", w.Code, w.Body.String())
	}

	var loginResp domain.LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	if loginResp.Tokens.AccessToken == "" || loginResp.Tokens.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens after verified login")
	}

	// Step 7: Access protected logout route with the issued access token -> 200 OK
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Tokens.AccessToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on protected logout route, got %d: %s", w.Code, w.Body.String())
	}
}
