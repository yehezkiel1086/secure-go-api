package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/handler"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

type mockAuthService struct {
	loginFn        func(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)
	refreshTokenFn func(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error)
	logoutFn       func(ctx context.Context, rawRefreshToken string) error
}

var _ port.AuthService = (*mockAuthService)(nil)

func (m *mockAuthService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, req)
	}
	return nil, domain.ErrInvalidCredentials
}

func (m *mockAuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error) {
	if m.refreshTokenFn != nil {
		return m.refreshTokenFn(ctx, rawRefreshToken)
	}
	return nil, domain.ErrInvalidToken
}

func (m *mockAuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if m.logoutFn != nil {
		return m.logoutFn(ctx, rawRefreshToken)
	}
	return nil
}

func setupAuthTestRouter(authSvc port.AuthService, userSvc port.UserService) *handler.Router {
	gin.SetMode(gin.TestMode)
	conf := &config.Container{
		App: &config.App{Name: "test-app", Env: "test"},
		JWT: testJWTConfig,
	}
	userH := handler.NewUserHandler(userSvc)
	authH := handler.NewAuthHandler(authSvc, conf.JWT, conf.App)
	jobH := handler.NewJobHandler(nil)
	return handler.NewRouter(conf, userH, authH, jobH)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	authSvc := &mockAuthService{
		loginFn: func(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
			return &domain.LoginResponse{
				User: &domain.UserResponse{
					ID:    "1",
					Name:  "Bob",
					Email: req.Email,
					Role:  domain.RoleUser,
				},
				Tokens: &domain.TokenPair{
					AccessToken:           "access123",
					RefreshToken:          "refresh123",
					AccessTokenExpiresAt:  time.Now().Add(15 * time.Minute),
					RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				},
			}, nil
		},
	}

	router := setupAuthTestRouter(authSvc, &mockUserService{})

	payload := domain.LoginRequest{
		Email:    "bob@example.com",
		Password: "correctpassword",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// Verify cookies were set
	cookies := w.Result().Cookies()
	var hasAccessCookie, hasRefreshCookie bool
	for _, c := range cookies {
		if c.Name == "access_token" && c.Value == "access123" {
			hasAccessCookie = true
			if !c.HttpOnly {
				t.Error("expected access_token cookie to be HttpOnly")
			}
		}
		if c.Name == "refresh_token" && c.Value == "refresh123" {
			hasRefreshCookie = true
			if !c.HttpOnly {
				t.Error("expected refresh_token cookie to be HttpOnly")
			}
			if c.Path != "/api/v1/refresh" {
				t.Errorf("expected refresh cookie path /api/v1/refresh, got %s", c.Path)
			}
		}
	}

	if !hasAccessCookie || !hasRefreshCookie {
		t.Fatal("expected both access and refresh cookies to be set")
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	authSvc := &mockAuthService{
		loginFn: func(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
			return nil, domain.ErrInvalidCredentials
		},
	}

	router := setupAuthTestRouter(authSvc, &mockUserService{})

	payload := domain.LoginRequest{
		Email:    "bob@example.com",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestAuthHandler_Login_EmailNotVerified(t *testing.T) {
	authSvc := &mockAuthService{
		loginFn: func(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
			return nil, domain.ErrEmailNotVerified
		},
	}

	router := setupAuthTestRouter(authSvc, &mockUserService{})

	payload := domain.LoginRequest{
		Email:    "bob@example.com",
		Password: "correctpassword",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}
}


func TestAuthHandler_RefreshToken_ReuseDetection(t *testing.T) {
	authSvc := &mockAuthService{
		refreshTokenFn: func(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error) {
			return nil, domain.ErrTokenReuse
		},
	}

	router := setupAuthTestRouter(authSvc, &mockUserService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "stolen-reused-token"})
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on reuse detection, got %d", w.Code)
	}

	// Verify cookies were cleared (MaxAge < 0)
	for _, c := range w.Result().Cookies() {
		if c.MaxAge > 0 {
			t.Errorf("expected cookie %s to be cleared, got maxage %d", c.Name, c.MaxAge)
		}
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	authSvc := &mockAuthService{
		logoutFn: func(ctx context.Context, rawRefreshToken string) error {
			return nil
		},
	}

	router := setupAuthTestRouter(authSvc, &mockUserService{})

	userToken, _ := util.GenerateToken(testJWTConfig, util.TokenAccess, &domain.User{
		ID:   pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		Name: "Test User",
		Role: domain.RoleUser,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "token-to-logout"})
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on logout, got %d", w.Code)
	}

	for _, c := range w.Result().Cookies() {
		if c.MaxAge > 0 {
			t.Errorf("expected cookie %s to be cleared on logout, got maxage %d", c.Name, c.MaxAge)
		}
	}
}

func TestMiddleware_RBAC(t *testing.T) {
	router := setupAuthTestRouter(&mockAuthService{}, &mockUserService{
		getUsersFn: func(ctx context.Context, page, pageSize int32) (*domain.PaginatedUsersResponse, error) {
			return &domain.PaginatedUsersResponse{}, nil
		},
	})

	regularUserToken, _ := util.GenerateToken(testJWTConfig, util.TokenAccess, &domain.User{
		ID:   pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
		Name: "Regular User",
		Role: domain.RoleUser, // 2001
	})

	adminToken, _ := util.GenerateToken(testJWTConfig, util.TokenAccess, &domain.User{
		ID:   pgtype.UUID{Bytes: [16]byte{3}, Valid: true},
		Name: "Admin User",
		Role: domain.RoleAdmin, // 5150
	})

	// 1. Regular user accessing admin-only endpoint /api/v1/users -> 403 Forbidden
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req1.Header.Set("Authorization", "Bearer "+regularUserToken)
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for regular user on admin route, got %d", w1.Code)
	}

	// 2. Admin accessing admin-only endpoint /api/v1/users -> 200 OK
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin user, got %d", w2.Code)
	}

	// 3. Unauthenticated request to /api/v1/users -> 401 Unauthorized
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", w3.Code)
	}
}
