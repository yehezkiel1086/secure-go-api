package port

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
)

type JobRepository interface {
	CreateJob(ctx context.Context, job *domain.Job) (*domain.Job, error)
	GetJobByID(ctx context.Context, id pgtype.UUID) (*domain.Job, error)
	ListJobs(ctx context.Context, limit, offset int32) ([]*domain.Job, error)
	CountJobs(ctx context.Context) (int64, error)
	SearchJobs(ctx context.Context, query string, limit int32) ([]*domain.Job, error)
	UpdateJob(ctx context.Context, job *domain.Job) (*domain.Job, error)
	DeleteJob(ctx context.Context, id pgtype.UUID) error
}

type JobService interface {
	CreateJob(ctx context.Context, adminID pgtype.UUID, req *domain.CreateJobRequest) (*domain.JobResponse, error)
	GetJobByID(ctx context.Context, id pgtype.UUID) (*domain.JobResponse, error)
	GetJobs(ctx context.Context, page, pageSize int32, query string) (*domain.PaginatedJobsResponse, error)
	UpdateJob(ctx context.Context, id pgtype.UUID, req *domain.UpdateJobRequest) (*domain.JobResponse, error)
	DeleteJob(ctx context.Context, id pgtype.UUID) error
}
