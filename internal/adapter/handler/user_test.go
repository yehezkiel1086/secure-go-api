package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/handler"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/util"
)

type mockUserService struct {
	registerUserFn          func(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error)
	getUserByIDFn           func(ctx context.Context, id pgtype.UUID) (*domain.UserResponse, error)
	getUsersFn              func(ctx context.Context, page, pageSize int32) (*domain.PaginatedUsersResponse, error)
	updateUserNameFn        func(ctx context.Context, id pgtype.UUID, req *domain.UpdateUserNameRequest) (*domain.UserResponse, error)
	verifyEmailFn           func(ctx context.Context, token string) error
	resendVerificationEmailFn func(ctx context.Context, email string) error
}

var _ port.UserService = (*mockUserService)(nil)

func (m *mockUserService) RegisterUser(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error) {
	if m.registerUserFn != nil {
		return m.registerUserFn(ctx, req)
	}
	return nil, nil
}

func (m *mockUserService) GetUserByID(ctx context.Context, id pgtype.UUID) (*domain.UserResponse, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserService) GetUsers(ctx context.Context, page, pageSize int32) (*domain.PaginatedUsersResponse, error) {
	if m.getUsersFn != nil {
		return m.getUsersFn(ctx, page, pageSize)
	}
	return nil, nil
}

func (m *mockUserService) UpdateUserName(ctx context.Context, id pgtype.UUID, req *domain.UpdateUserNameRequest) (*domain.UserResponse, error) {
	if m.updateUserNameFn != nil {
		return m.updateUserNameFn(ctx, id, req)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserService) VerifyEmail(ctx context.Context, token string) error {
	if m.verifyEmailFn != nil {
		return m.verifyEmailFn(ctx, token)
	}
	return nil
}

func (m *mockUserService) ResendVerificationEmail(ctx context.Context, email string) error {
	if m.resendVerificationEmailFn != nil {
		return m.resendVerificationEmailFn(ctx, email)
	}
	return nil
}

var testJWTConfig = &config.JWT{
	AccessTokenSecret:    "testaccesssecret12345678901234567890",
	RefreshTokenSecret:   "testrefreshsecret12345678901234567890",
	AccessTokenDuration:  "15",
	RefreshTokenDuration: "7",
}

func setupTestRouter(svc port.UserService) *handler.Router {
	gin.SetMode(gin.TestMode)
	conf := &config.Container{
		App: &config.App{Name: "test-app", Env: "test"},
		JWT: testJWTConfig,
	}
	userH := handler.NewUserHandler(svc)
	authH := handler.NewAuthHandler(nil, conf.JWT, conf.App)
	jobH := handler.NewJobHandler(nil)
	return handler.NewRouter(conf, userH, authH, jobH)
}

func TestHandler_HealthCheck(t *testing.T) {
	router := setupTestRouter(&mockUserService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandler_SwaggerUI(t *testing.T) {
	router := setupTestRouter(&mockUserService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	req.RequestURI = "/swagger/index.html"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for swagger UI, got %d", w.Code)
	}
}

func TestHandler_SwaggerDocJSON(t *testing.T) {
	router := setupTestRouter(&mockUserService{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	req.RequestURI = "/swagger/doc.json"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for swagger doc.json, got %d", w.Code)
	}
}

func TestHandler_RegisterUser_Success(t *testing.T) {
	svc := &mockUserService{
		registerUserFn: func(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error) {
			return &domain.UserResponse{
				ID:    "123",
				Name:  req.Name,
				Email: req.Email,
				Role:  domain.RoleUser,
			}, nil
		},
	}
	router := setupTestRouter(svc)

	payload := domain.RegisterUserRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "strongpassword123",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_RegisterUser_Conflict(t *testing.T) {
	svc := &mockUserService{
		registerUserFn: func(ctx context.Context, req *domain.RegisterUserRequest) (*domain.UserResponse, error) {
			return nil, domain.ErrEmailAlreadyExists
		},
	}
	router := setupTestRouter(svc)

	payload := domain.RegisterUserRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "strongpassword123",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", w.Code)
	}
}

func TestHandler_ConfirmEmail(t *testing.T) {
	svc := &mockUserService{
		verifyEmailFn: func(ctx context.Context, token string) error {
			return nil
		},
	}
	router := setupTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/confirm-email?token=valid-token", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestHandler_GetUserByID_InvalidUUID(t *testing.T) {
	router := setupTestRouter(&mockUserService{})

	adminToken, err := util.GenerateToken(testJWTConfig, util.TokenAccess, &domain.User{
		Name:  "Admin",
		Email: "admin@example.com",
		Role:  domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}
}

func TestHandler_ResendVerification(t *testing.T) {
	svc := &mockUserService{
		resendVerificationEmailFn: func(ctx context.Context, email string) error {
			return nil
		},
	}
	router := setupTestRouter(svc)

	payload := domain.ResendVerificationRequest{
		Email: "bob@example.com",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/resend-verification", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
}

