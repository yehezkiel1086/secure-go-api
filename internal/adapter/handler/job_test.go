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

type mockJobService struct {
	createJobFn  func(ctx context.Context, adminID pgtype.UUID, req *domain.CreateJobRequest) (*domain.JobResponse, error)
	getJobByIDFn func(ctx context.Context, id pgtype.UUID) (*domain.JobResponse, error)
	getJobsFn    func(ctx context.Context, page, pageSize int32, query string) (*domain.PaginatedJobsResponse, error)
	updateJobFn  func(ctx context.Context, id pgtype.UUID, req *domain.UpdateJobRequest) (*domain.JobResponse, error)
	deleteJobFn  func(ctx context.Context, id pgtype.UUID) error
}

var _ port.JobService = (*mockJobService)(nil)

func (m *mockJobService) CreateJob(ctx context.Context, adminID pgtype.UUID, req *domain.CreateJobRequest) (*domain.JobResponse, error) {
	if m.createJobFn != nil {
		return m.createJobFn(ctx, adminID, req)
	}
	return nil, nil
}

func (m *mockJobService) GetJobByID(ctx context.Context, id pgtype.UUID) (*domain.JobResponse, error) {
	if m.getJobByIDFn != nil {
		return m.getJobByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockJobService) GetJobs(ctx context.Context, page, pageSize int32, query string) (*domain.PaginatedJobsResponse, error) {
	if m.getJobsFn != nil {
		return m.getJobsFn(ctx, page, pageSize, query)
	}
	return &domain.PaginatedJobsResponse{Jobs: []*domain.JobResponse{}}, nil
}

func (m *mockJobService) UpdateJob(ctx context.Context, id pgtype.UUID, req *domain.UpdateJobRequest) (*domain.JobResponse, error) {
	if m.updateJobFn != nil {
		return m.updateJobFn(ctx, id, req)
	}
	return nil, domain.ErrNotFound
}

func (m *mockJobService) DeleteJob(ctx context.Context, id pgtype.UUID) error {
	if m.deleteJobFn != nil {
		return m.deleteJobFn(ctx, id)
	}
	return nil
}

func setupJobTestRouter(jobSvc port.JobService) *handler.Router {
	gin.SetMode(gin.TestMode)
	conf := &config.Container{
		App: &config.App{Name: "test-app", Env: "test"},
		JWT: testJWTConfig,
	}
	userH := handler.NewUserHandler(&mockUserService{})
	authH := handler.NewAuthHandler(nil, conf.JWT, conf.App)
	jobH := handler.NewJobHandler(jobSvc)
	return handler.NewRouter(conf, userH, authH, jobH)
}

func generateTestToken(t *testing.T, role int32) string {
	t.Helper()
	var userID pgtype.UUID
	_ = userID.Scan("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	token, err := util.GenerateToken(testJWTConfig, util.TokenAccess, &domain.User{
		ID:    userID,
		Name:  "Test User",
		Email: "test@example.com",
		Role:  role,
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}

func TestJobHandler_GetJobs(t *testing.T) {
	t.Run("unauthenticated returns 401", func(t *testing.T) {
		router := setupJobTestRouter(&mockJobService{})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("regular user role returns 200", func(t *testing.T) {
		userToken := generateTestToken(t, domain.RoleUser)
		svc := &mockJobService{
			getJobsFn: func(ctx context.Context, page, pageSize int32, query string) (*domain.PaginatedJobsResponse, error) {
				return &domain.PaginatedJobsResponse{
					Jobs: []*domain.JobResponse{
						{ID: "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", Title: "Backend Engineer"},
					},
					Total:      1,
					Page:       1,
					PageSize:   10,
					TotalPages: 1,
				}, nil
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs?page=1&page_size=10", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})

	t.Run("admin role returns 200", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			getJobsFn: func(ctx context.Context, page, pageSize int32, query string) (*domain.PaginatedJobsResponse, error) {
				return &domain.PaginatedJobsResponse{
					Jobs:       []*domain.JobResponse{},
					Total:      0,
					Page:       1,
					PageSize:   10,
					TotalPages: 1,
				}, nil
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})
}

func TestJobHandler_GetJobByID(t *testing.T) {
	userToken := generateTestToken(t, domain.RoleUser)

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		router := setupJobTestRouter(&mockJobService{})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs/invalid-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("job not found returns 404", func(t *testing.T) {
		svc := &mockJobService{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.JobResponse, error) {
				return nil, domain.ErrNotFound
			},
		}
		router := setupJobTestRouter(svc)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})

	t.Run("found returns 200", func(t *testing.T) {
		svc := &mockJobService{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.JobResponse, error) {
				return &domain.JobResponse{
					ID:    "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
					Title: "Security Engineer",
				}, nil
			},
		}
		router := setupJobTestRouter(svc)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})
}

func TestJobHandler_CreateJob_RBAC(t *testing.T) {
	payload := domain.CreateJobRequest{
		Title:       "Backend Engineer",
		Description: "Develop Go microservices with Hexagonal Architecture",
		Company:     "Secure Corp",
		Location:    "Remote",
	}
	body, _ := json.Marshal(payload)

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		router := setupJobTestRouter(&mockJobService{})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("regular user returns 403 Forbidden", func(t *testing.T) {
		userToken := generateTestToken(t, domain.RoleUser)
		router := setupJobTestRouter(&mockJobService{})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for regular user, got %d", w.Code)
		}
	})

	t.Run("admin role creates job successfully returns 201", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			createJobFn: func(ctx context.Context, adminID pgtype.UUID, req *domain.CreateJobRequest) (*domain.JobResponse, error) {
				return &domain.JobResponse{
					ID:        "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
					Title:     req.Title,
					Company:   req.Company,
					Location:  req.Location,
					CreatedBy: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
				}, nil
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for admin, got %d", w.Code)
		}
	})

	t.Run("admin role with invalid salary range returns 400", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			createJobFn: func(ctx context.Context, adminID pgtype.UUID, req *domain.CreateJobRequest) (*domain.JobResponse, error) {
				return nil, domain.ErrInvalidSalaryRange
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid salary range, got %d", w.Code)
		}
	})

	t.Run("admin role with malformed body returns 400", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		router := setupJobTestRouter(&mockJobService{})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader([]byte(`{"title": ""}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}

func TestJobHandler_UpdateJob_RBAC(t *testing.T) {
	updateTitle := "Updated Title"
	payload := domain.UpdateJobRequest{
		Title: &updateTitle,
	}
	body, _ := json.Marshal(payload)

	t.Run("regular user returns 403 Forbidden", func(t *testing.T) {
		userToken := generateTestToken(t, domain.RoleUser)
		router := setupJobTestRouter(&mockJobService{})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+userToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("admin success returns 200", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			updateJobFn: func(ctx context.Context, id pgtype.UUID, req *domain.UpdateJobRequest) (*domain.JobResponse, error) {
				return &domain.JobResponse{
					ID:    "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
					Title: *req.Title,
				}, nil
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})

	t.Run("admin invalid salary range returns 400", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			updateJobFn: func(ctx context.Context, id pgtype.UUID, req *domain.UpdateJobRequest) (*domain.JobResponse, error) {
				return nil, domain.ErrInvalidSalaryRange
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("admin job not found returns 404", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			updateJobFn: func(ctx context.Context, id pgtype.UUID, req *domain.UpdateJobRequest) (*domain.JobResponse, error) {
				return nil, domain.ErrNotFound
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})
}

func TestJobHandler_DeleteJob_RBAC(t *testing.T) {
	t.Run("regular user returns 403 Forbidden", func(t *testing.T) {
		userToken := generateTestToken(t, domain.RoleUser)
		router := setupJobTestRouter(&mockJobService{})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("admin success returns 200", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			deleteJobFn: func(ctx context.Context, id pgtype.UUID) error {
				return nil
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})

	t.Run("admin not found returns 404", func(t *testing.T) {
		adminToken := generateTestToken(t, domain.RoleAdmin)
		svc := &mockJobService{
			deleteJobFn: func(ctx context.Context, id pgtype.UUID) error {
				return domain.ErrNotFound
			},
		}
		router := setupJobTestRouter(svc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/jobs/b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})
}
