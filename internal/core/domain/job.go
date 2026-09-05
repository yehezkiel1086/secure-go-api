package domain

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Job struct {
	ID            pgtype.UUID
	Title         string
	Description   string
	Company       string
	Location      string
	SalaryMin     pgtype.Int8
	SalaryMax     pgtype.Int8
	CreatedBy     pgtype.UUID
	CreatedByName string
	CreatedAt     pgtype.Timestamptz
	UpdatedAt     pgtype.Timestamptz
}

type CreateJobRequest struct {
	Title       string `json:"title" binding:"required,min=3,max=150"`
	Description string `json:"description" binding:"required,min=10,max=10000"`
	Company     string `json:"company" binding:"required,min=2,max=100"`
	Location    string `json:"location" binding:"required,min=2,max=100"`
	SalaryMin   *int64 `json:"salary_min,omitempty" binding:"omitempty,gte=0"`
	SalaryMax   *int64 `json:"salary_max,omitempty" binding:"omitempty,gte=0"`
}

type UpdateJobRequest struct {
	Title       *string `json:"title,omitempty" binding:"omitempty,min=3,max=150"`
	Description *string `json:"description,omitempty" binding:"omitempty,min=10,max=10000"`
	Company     *string `json:"company,omitempty" binding:"omitempty,min=2,max=100"`
	Location    *string `json:"location,omitempty" binding:"omitempty,min=2,max=100"`
	SalaryMin   *int64  `json:"salary_min,omitempty" binding:"omitempty,gte=0"`
	SalaryMax   *int64  `json:"salary_max,omitempty" binding:"omitempty,gte=0"`
}

type JobResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	Company       string `json:"company"`
	Location      string `json:"location"`
	SalaryMin     *int64 `json:"salary_min,omitempty"`
	SalaryMax     *int64 `json:"salary_max,omitempty"`
	CreatedBy     string `json:"created_by"`
	CreatedByName string `json:"created_by_name,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type PaginatedJobsResponse struct {
	Jobs       []*JobResponse `json:"jobs"`
	Total      int64          `json:"total"`
	Page       int32          `json:"page"`
	PageSize   int32          `json:"page_size"`
	TotalPages int32          `json:"total_pages"`
}

func (j *Job) ToResponse() *JobResponse {
	var idStr string
	if j.ID.Valid {
		b := j.ID.Bytes
		idStr = fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
			b[0], b[1], b[2], b[3],
			b[4], b[5],
			b[6], b[7],
			b[8], b[9],
			b[10], b[11], b[12], b[13], b[14], b[15],
		)
	}

	var createdByStr string
	if j.CreatedBy.Valid {
		b := j.CreatedBy.Bytes
		createdByStr = fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
			b[0], b[1], b[2], b[3],
			b[4], b[5],
			b[6], b[7],
			b[8], b[9],
			b[10], b[11], b[12], b[13], b[14], b[15],
		)
	}

	var salaryMin, salaryMax *int64
	if j.SalaryMin.Valid {
		val := j.SalaryMin.Int64
		salaryMin = &val
	}
	if j.SalaryMax.Valid {
		val := j.SalaryMax.Int64
		salaryMax = &val
	}

	var createdAtStr, updatedAtStr string
	if j.CreatedAt.Valid {
		createdAtStr = j.CreatedAt.Time.Format(time.RFC3339)
	}
	if j.UpdatedAt.Valid {
		updatedAtStr = j.UpdatedAt.Time.Format(time.RFC3339)
	}

	return &JobResponse{
		ID:            idStr,
		Title:         j.Title,
		Description:   j.Description,
		Company:       j.Company,
		Location:      j.Location,
		SalaryMin:     salaryMin,
		SalaryMax:     salaryMax,
		CreatedBy:     createdByStr,
		CreatedByName: j.CreatedByName,
		CreatedAt:     createdAtStr,
		UpdatedAt:     updatedAtStr,
	}
}
