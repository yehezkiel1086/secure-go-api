package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yehezkiel1086/secure-go-api/internal/core/domain"
	"github.com/yehezkiel1086/secure-go-api/internal/core/port"
)

var _ port.JobService = (*JobService)(nil)

type JobService struct {
	jobRepo port.JobRepository
}

func NewJobService(jobRepo port.JobRepository) *JobService {
	return &JobService{
		jobRepo: jobRepo,
	}
}

func (s *JobService) CreateJob(ctx context.Context, adminID pgtype.UUID, req *domain.CreateJobRequest) (*domain.JobResponse, error) {
	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	company := strings.TrimSpace(req.Company)
	location := strings.TrimSpace(req.Location)

	// validate salary range
	if req.SalaryMin != nil && req.SalaryMax != nil {
		if *req.SalaryMin > *req.SalaryMax {
			return nil, domain.ErrInvalidSalaryRange
		}
	}

	var salaryMin, salaryMax pgtype.Int8
	if req.SalaryMin != nil {
		salaryMin = pgtype.Int8{Int64: *req.SalaryMin, Valid: true}
	}
	if req.SalaryMax != nil {
		salaryMax = pgtype.Int8{Int64: *req.SalaryMax, Valid: true}
	}

	job := &domain.Job{
		Title:       title,
		Description: description,
		Company:     company,
		Location:    location,
		SalaryMin:   salaryMin,
		SalaryMax:   salaryMax,
		CreatedBy:   adminID,
	}

	created, err := s.jobRepo.CreateJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("creating job: %w", err)
	}

	return created.ToResponse(), nil
}

func (s *JobService) GetJobByID(ctx context.Context, id pgtype.UUID) (*domain.JobResponse, error) {
	job, err := s.jobRepo.GetJobByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return job.ToResponse(), nil
}

func (s *JobService) GetJobs(ctx context.Context, page, pageSize int32, query string) (*domain.PaginatedJobsResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query = strings.TrimSpace(query)

	var (
		jobs       []*domain.Job
		total      int64
		totalPages int32
		err        error
	)

	if query != "" {
		// full-text search
		jobs, err = s.jobRepo.SearchJobs(ctx, query, pageSize)
		if err != nil {
			return nil, fmt.Errorf("searching jobs: %w", err)
		}
		total = int64(len(jobs))
		totalPages = 1
	} else {
		// offset pagination
		offset := (page - 1) * pageSize
		limit := pageSize

		jobs, err = s.jobRepo.ListJobs(ctx, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("listing jobs: %w", err)
		}

		total, err = s.jobRepo.CountJobs(ctx)
		if err != nil {
			return nil, fmt.Errorf("counting jobs: %w", err)
		}

		totalPages = int32(math.Ceil(float64(total) / float64(pageSize)))
		if totalPages == 0 {
			totalPages = 1
		}
	}

	resJobs := make([]*domain.JobResponse, len(jobs))
	for i, j := range jobs {
		resJobs[i] = j.ToResponse()
	}

	return &domain.PaginatedJobsResponse{
		Jobs:       resJobs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *JobService) UpdateJob(ctx context.Context, id pgtype.UUID, req *domain.UpdateJobRequest) (*domain.JobResponse, error) {
	existing, err := s.jobRepo.GetJobByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t != "" {
			existing.Title = t
		}
	}
	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		if d != "" {
			existing.Description = d
		}
	}
	if req.Company != nil {
		c := strings.TrimSpace(*req.Company)
		if c != "" {
			existing.Company = c
		}
	}
	if req.Location != nil {
		l := strings.TrimSpace(*req.Location)
		if l != "" {
			existing.Location = l
		}
	}
	if req.SalaryMin != nil {
		existing.SalaryMin = pgtype.Int8{Int64: *req.SalaryMin, Valid: true}
	}
	if req.SalaryMax != nil {
		existing.SalaryMax = pgtype.Int8{Int64: *req.SalaryMax, Valid: true}
	}

	// validate salary range
	if existing.SalaryMin.Valid && existing.SalaryMax.Valid {
		if existing.SalaryMin.Int64 > existing.SalaryMax.Int64 {
			return nil, domain.ErrInvalidSalaryRange
		}
	}

	updated, err := s.jobRepo.UpdateJob(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("updating job: %w", err)
	}

	return updated.ToResponse(), nil
}

func (s *JobService) DeleteJob(ctx context.Context, id pgtype.UUID) error {
	if _, err := s.jobRepo.GetJobByID(ctx, id); err != nil {
		return err
	}

	if err := s.jobRepo.DeleteJob(ctx, id); err != nil {
		return fmt.Errorf("deleting job: %w", err)
	}

	return nil
}
