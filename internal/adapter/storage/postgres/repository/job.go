package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres/sqlc"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

var _ port.JobRepository = (*JobRepository)(nil)

type JobRepository struct {
	queries *sqlc.Queries
}

func NewJobRepository(queries *sqlc.Queries) *JobRepository {
	return &JobRepository{
		queries: queries,
	}
}

func NewJobRepositoryWithDB(db sqlc.DBTX) *JobRepository {
	return &JobRepository{
		queries: sqlc.New(db),
	}
}

func (r *JobRepository) CreateJob(ctx context.Context, job *domain.Job) (*domain.Job, error) {
	row, err := r.queries.CreateJob(ctx, sqlc.CreateJobParams{
		Title:       job.Title,
		Description: job.Description,
		Company:     job.Company,
		Location:    job.Location,
		SalaryMin:   job.SalaryMin,
		SalaryMax:   job.SalaryMax,
		CreatedBy:   job.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	return &domain.Job{
		ID:          row.ID,
		Title:       row.Title,
		Description: row.Description,
		Company:     row.Company,
		Location:    row.Location,
		SalaryMin:   row.SalaryMin,
		SalaryMax:   row.SalaryMax,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *JobRepository) GetJobByID(ctx context.Context, id pgtype.UUID) (*domain.Job, error) {
	row, err := r.queries.GetJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get job by id: %w", err)
	}

	return &domain.Job{
		ID:            row.ID,
		Title:         row.Title,
		Description:   row.Description,
		Company:       row.Company,
		Location:      row.Location,
		SalaryMin:     row.SalaryMin,
		SalaryMax:     row.SalaryMax,
		CreatedBy:     row.CreatedBy,
		CreatedByName: row.CreatedByName,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r *JobRepository) ListJobs(ctx context.Context, limit, offset int32) ([]*domain.Job, error) {
	rows, err := r.queries.ListJobsOffset(ctx, sqlc.ListJobsOffsetParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	jobs := make([]*domain.Job, len(rows))
	for i, row := range rows {
		jobs[i] = &domain.Job{
			ID:        row.ID,
			Title:     row.Title,
			Company:   row.Company,
			Location:  row.Location,
			SalaryMin: row.SalaryMin,
			SalaryMax: row.SalaryMax,
			CreatedAt: row.CreatedAt,
		}
	}

	return jobs, nil
}

func (r *JobRepository) CountJobs(ctx context.Context) (int64, error) {
	count, err := r.queries.CountJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("count jobs: %w", err)
	}
	return count, nil
}

func (r *JobRepository) SearchJobs(ctx context.Context, query string, limit int32) ([]*domain.Job, error) {
	rows, err := r.queries.SearchJobs(ctx, sqlc.SearchJobsParams{
		PlaintoTsquery: query,
		Limit:          limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search jobs: %w", err)
	}

	jobs := make([]*domain.Job, len(rows))
	for i, row := range rows {
		jobs[i] = &domain.Job{
			ID:        row.ID,
			Title:     row.Title,
			Company:   row.Company,
			Location:  row.Location,
			SalaryMin: row.SalaryMin,
			SalaryMax: row.SalaryMax,
			CreatedAt: row.CreatedAt,
		}
	}

	return jobs, nil
}

func (r *JobRepository) UpdateJob(ctx context.Context, job *domain.Job) (*domain.Job, error) {
	row, err := r.queries.UpdateJob(ctx, sqlc.UpdateJobParams{
		ID:          job.ID,
		Title:       job.Title,
		Description: job.Description,
		Company:     job.Company,
		Location:    job.Location,
		SalaryMin:   job.SalaryMin,
		SalaryMax:   job.SalaryMax,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update job: %w", err)
	}

	return &domain.Job{
		ID:          row.ID,
		Title:       row.Title,
		Description: row.Description,
		Company:     row.Company,
		Location:    row.Location,
		SalaryMin:   row.SalaryMin,
		SalaryMax:   row.SalaryMax,
		CreatedBy:   row.CreatedBy,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *JobRepository) DeleteJob(ctx context.Context, id pgtype.UUID) error {
	err := r.queries.DeleteJob(ctx, id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	return nil
}
