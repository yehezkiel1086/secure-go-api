package domain

import "github.com/jackc/pgx/v5/pgtype"

// Write operations (POST, DELETE) are scoped to Admin role (5150) only.
type Job struct {
	ID          pgtype.UUID
	Title       string
	Description string
	Company     string
	Location    string
	SalaryMin   pgtype.Int8
	SalaryMax   pgtype.Int8
	// Admin user who created the listing
	CreatedBy pgtype.UUID
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}
