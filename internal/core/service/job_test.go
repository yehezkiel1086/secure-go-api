package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
	"github.com/yehezkiel1086/secure-go-api/internal/core/service"
)

type mockJobRepo struct {
	createJobFn  func(ctx context.Context, job *domain.Job) (*domain.Job, error)
	getJobByIDFn func(ctx context.Context, id pgtype.UUID) (*domain.Job, error)
	listJobsFn   func(ctx context.Context, limit, offset int32) ([]*domain.Job, error)
	searchJobsFn func(ctx context.Context, query string, limit int32) ([]*domain.Job, error)
	countJobsFn  func(ctx context.Context) (int64, error)
	updateJobFn  func(ctx context.Context, job *domain.Job) (*domain.Job, error)
	deleteJobFn  func(ctx context.Context, id pgtype.UUID) error
}

var _ port.JobRepository = (*mockJobRepo)(nil)

func (m *mockJobRepo) CreateJob(ctx context.Context, job *domain.Job) (*domain.Job, error) {
	if m.createJobFn != nil {
		return m.createJobFn(ctx, job)
	}
	return job, nil
}

func (m *mockJobRepo) GetJobByID(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
	if m.getJobByIDFn != nil {
		return m.getJobByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockJobRepo) ListJobs(ctx context.Context, limit, offset int32) ([]*domain.Job, error) {
	if m.listJobsFn != nil {
		return m.listJobsFn(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockJobRepo) SearchJobs(ctx context.Context, query string, limit int32) ([]*domain.Job, error) {
	if m.searchJobsFn != nil {
		return m.searchJobsFn(ctx, query, limit)
	}
	return nil, nil
}

func (m *mockJobRepo) CountJobs(ctx context.Context) (int64, error) {
	if m.countJobsFn != nil {
		return m.countJobsFn(ctx)
	}
	return 0, nil
}

func (m *mockJobRepo) UpdateJob(ctx context.Context, job *domain.Job) (*domain.Job, error) {
	if m.updateJobFn != nil {
		return m.updateJobFn(ctx, job)
	}
	return job, nil
}

func (m *mockJobRepo) DeleteJob(ctx context.Context, id pgtype.UUID) error {
	if m.deleteJobFn != nil {
		return m.deleteJobFn(ctx, id)
	}
	return nil
}

func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func TestJobService_CreateJob(t *testing.T) {
	adminID := parseUUID("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	t.Run("success", func(t *testing.T) {
		minSalary := int64(5000)
		maxSalary := int64(10000)
		repo := &mockJobRepo{
			createJobFn: func(ctx context.Context, job *domain.Job) (*domain.Job, error) {
				if job.Title != "Software Engineer" {
					t.Errorf("expected trimmed title, got %q", job.Title)
				}
				if job.CreatedBy != adminID {
					t.Errorf("expected created_by to match adminID")
				}
				job.ID = parseUUID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")
				job.CreatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
				job.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
				return job, nil
			},
		}

		svc := service.NewJobService(repo)
		req := &domain.CreateJobRequest{
			Title:       "  Software Engineer  ",
			Description: "Develop Go microservices",
			Company:     "Tech Corp",
			Location:    "Remote",
			SalaryMin:   &minSalary,
			SalaryMax:   &maxSalary,
		}

		res, err := svc.CreateJob(context.Background(), adminID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Title != "Software Engineer" {
			t.Errorf("expected Software Engineer, got %s", res.Title)
		}
		if res.SalaryMin == nil || *res.SalaryMin != 5000 {
			t.Errorf("expected salary min 5000, got %v", res.SalaryMin)
		}
	})

	t.Run("invalid salary range - min greater than max", func(t *testing.T) {
		minSalary := int64(12000)
		maxSalary := int64(5000)
		repo := &mockJobRepo{}
		svc := service.NewJobService(repo)

		req := &domain.CreateJobRequest{
			Title:       "Software Engineer",
			Description: "Go dev",
			Company:     "Tech Corp",
			Location:    "Remote",
			SalaryMin:   &minSalary,
			SalaryMax:   &maxSalary,
		}

		_, err := svc.CreateJob(context.Background(), adminID, req)
		if !errors.Is(err, domain.ErrInvalidSalaryRange) {
			t.Fatalf("expected ErrInvalidSalaryRange, got %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockJobRepo{
			createJobFn: func(ctx context.Context, job *domain.Job) (*domain.Job, error) {
				return nil, errors.New("db error")
			},
		}
		svc := service.NewJobService(repo)
		req := &domain.CreateJobRequest{
			Title:       "Software Engineer",
			Description: "Go dev",
			Company:     "Tech Corp",
			Location:    "Remote",
		}

		_, err := svc.CreateJob(context.Background(), adminID, req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestJobService_GetJobByID(t *testing.T) {
	jobID := parseUUID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")

	t.Run("found", func(t *testing.T) {
		repo := &mockJobRepo{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
				return &domain.Job{
					ID:        id,
					Title:     "DevOps Engineer",
					Company:   "Cloud Corp",
					CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
				}, nil
			},
		}
		svc := service.NewJobService(repo)
		res, err := svc.GetJobByID(context.Background(), jobID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Title != "DevOps Engineer" {
			t.Errorf("expected DevOps Engineer, got %s", res.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockJobRepo{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
				return nil, domain.ErrNotFound
			},
		}
		svc := service.NewJobService(repo)
		_, err := svc.GetJobByID(context.Background(), jobID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestJobService_GetJobs(t *testing.T) {
	t.Run("list without search query", func(t *testing.T) {
		repo := &mockJobRepo{
			listJobsFn: func(ctx context.Context, limit, offset int32) ([]*domain.Job, error) {
				if limit != 10 || offset != 0 {
					t.Errorf("expected limit 10, offset 0, got %d, %d", limit, offset)
				}
				return []*domain.Job{
					{
						ID:        parseUUID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"),
						Title:     "Job 1",
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					},
				}, nil
			},
			countJobsFn: func(ctx context.Context) (int64, error) {
				return 25, nil
			},
		}

		svc := service.NewJobService(repo)
		res, err := svc.GetJobs(context.Background(), 1, 10, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Total != 25 {
			t.Errorf("expected total count 25, got %d", res.Total)
		}
		if res.TotalPages != 3 {
			t.Errorf("expected 3 total pages for 25 items with page size 10, got %d", res.TotalPages)
		}
		if len(res.Jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(res.Jobs))
		}
	})

	t.Run("search query branch", func(t *testing.T) {
		repo := &mockJobRepo{
			searchJobsFn: func(ctx context.Context, query string, limit int32) ([]*domain.Job, error) {
				if query != "golang" {
					t.Errorf("expected golang, got %s", query)
				}
				return []*domain.Job{
					{
						ID:        parseUUID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22"),
						Title:     "Golang Developer",
						CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					},
				}, nil
			},
		}

		svc := service.NewJobService(repo)
		res, err := svc.GetJobs(context.Background(), 1, 10, "golang")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(res.Jobs))
		}
		if res.Jobs[0].Title != "Golang Developer" {
			t.Errorf("expected Golang Developer, got %s", res.Jobs[0].Title)
		}
	})

	t.Run("clamping default pagination", func(t *testing.T) {
		repo := &mockJobRepo{
			listJobsFn: func(ctx context.Context, limit, offset int32) ([]*domain.Job, error) {
				if limit != 100 || offset != 0 {
					t.Errorf("expected limit 100, offset 0; got limit=%d, offset=%d", limit, offset)
				}
				return []*domain.Job{}, nil
			},
			countJobsFn: func(ctx context.Context) (int64, error) {
				return 0, nil
			},
		}

		svc := service.NewJobService(repo)
		// Request negative page and excessive page size (> 100)
		res, err := svc.GetJobs(context.Background(), -1, 500, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Page != 1 {
			t.Errorf("expected page clamped to 1, got %d", res.Page)
		}
		if res.PageSize != 100 {
			t.Errorf("expected pageSize clamped to 100, got %d", res.PageSize)
		}
	})
}

func TestJobService_UpdateJob(t *testing.T) {
	jobID := parseUUID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")

	t.Run("success partial update", func(t *testing.T) {
		repo := &mockJobRepo{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
				return &domain.Job{
					ID:          id,
					Title:       "Old Title",
					Description: "Old Desc",
					Company:     "Old Company",
					Location:    "Old Location",
					SalaryMin:   pgtype.Int8{Int64: 3000, Valid: true},
					SalaryMax:   pgtype.Int8{Int64: 7000, Valid: true},
					CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
					UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
				}, nil
			},
			updateJobFn: func(ctx context.Context, job *domain.Job) (*domain.Job, error) {
				if job.Title != "New Title" {
					t.Errorf("expected New Title, got %s", job.Title)
				}
				return job, nil
			},
		}

		svc := service.NewJobService(repo)
		newTitle := "New Title"
		req := &domain.UpdateJobRequest{
			Title: &newTitle,
		}

		res, err := svc.UpdateJob(context.Background(), jobID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Title != "New Title" {
			t.Errorf("expected New Title, got %s", res.Title)
		}
	})

	t.Run("invalid salary range during update", func(t *testing.T) {
		repo := &mockJobRepo{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
				return &domain.Job{
					ID:        id,
					SalaryMin: pgtype.Int8{Int64: 3000, Valid: true},
					SalaryMax: pgtype.Int8{Int64: 7000, Valid: true},
				}, nil
			},
		}

		svc := service.NewJobService(repo)
		invalidMin := int64(8000) // greater than existing max (7000)
		req := &domain.UpdateJobRequest{
			SalaryMin: &invalidMin,
		}

		_, err := svc.UpdateJob(context.Background(), jobID, req)
		if !errors.Is(err, domain.ErrInvalidSalaryRange) {
			t.Fatalf("expected ErrInvalidSalaryRange, got %v", err)
		}
	})

	t.Run("job not found", func(t *testing.T) {
		repo := &mockJobRepo{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
				return nil, domain.ErrNotFound
			},
		}

		svc := service.NewJobService(repo)
		newTitle := "New Title"
		req := &domain.UpdateJobRequest{
			Title: &newTitle,
		}

		_, err := svc.UpdateJob(context.Background(), jobID, req)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestJobService_DeleteJob(t *testing.T) {
	jobID := parseUUID("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22")

	t.Run("success", func(t *testing.T) {
		deleted := false
		repo := &mockJobRepo{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
				return &domain.Job{ID: id}, nil
			},
			deleteJobFn: func(ctx context.Context, id pgtype.UUID) error {
				deleted = true
				return nil
			},
		}

		svc := service.NewJobService(repo)
		err := svc.DeleteJob(context.Background(), jobID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !deleted {
			t.Errorf("expected deleteJobFn to be called")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockJobRepo{
			getJobByIDFn: func(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
				return nil, domain.ErrNotFound
			},
		}

		svc := service.NewJobService(repo)
		err := svc.DeleteJob(context.Background(), jobID)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}
