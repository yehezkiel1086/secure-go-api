-- =============================================================================
-- queries/job.sql
-- Maps to: internal/core/port/job.go → JobRepository interface
-- Write operations (CreateJob, DeleteJob) are Admin-only — enforced at the
-- routing layer via RoleMiddleware(domain.AdminRole), not here.
-- =============================================================================


-- ---------------------------------------------------------------------------
-- CreateJob
-- Called by: JobService.CreateJob  (POST /jobs — AdminRole only)
-- created_by references the authenticated admin's user ID extracted from
-- the JWT claims in middleware.
-- ---------------------------------------------------------------------------
-- name: CreateJob :one
INSERT INTO jobs (
    title,
    description,
    company,
    location,
    salary_min,
    salary_max,
    created_by
)
VALUES (
    $1,  -- title
    $2,  -- description
    $3,  -- company
    $4,  -- location
    $5,  -- salary_min  (nullable)
    $6,  -- salary_max  (nullable)
    $7   -- created_by  (UUID of authenticated admin)
)
RETURNING
    id,
    title,
    description,
    company,
    location,
    salary_min,
    salary_max,
    created_by,
    created_at,
    updated_at;


-- ---------------------------------------------------------------------------
-- GetJobByID
-- Called by: JobService.GetJobById  (GET /jobs/:id — UserRole+)
-- ---------------------------------------------------------------------------
-- name: GetJobByID :one
SELECT
    j.id,
    j.title,
    j.description,
    j.company,
    j.location,
    j.salary_min,
    j.salary_max,
    j.created_by,
    u.name AS created_by_name,
    j.created_at,
    j.updated_at
FROM jobs j
JOIN users u ON u.id = j.created_by
WHERE j.id = $1
LIMIT 1;


-- ---------------------------------------------------------------------------
-- ListJobs
-- Called by: JobService.GetJobs  (GET /jobs — UserRole+)
-- Keyset pagination on (created_at, id) for stable ordering on large tables.
-- Pass cursor_created_at = 'infinity' and cursor_id = gen_random_uuid()
-- for the first page, then use the last row's values for subsequent pages.
-- ---------------------------------------------------------------------------
-- name: ListJobs :many
SELECT
    j.id,
    j.title,
    j.description,
    j.company,
    j.location,
    j.salary_min,
    j.salary_max,
    j.created_by,
    u.name AS created_by_name,
    j.created_at,
    j.updated_at
FROM jobs j
JOIN users u ON u.id = j.created_by
WHERE (j.created_at, j.id) < ($1, $2)   -- keyset cursor
ORDER BY
    j.created_at DESC,
    j.id          DESC
LIMIT $3;   -- page_size


-- ---------------------------------------------------------------------------
-- ListJobsOffset
-- Simple LIMIT/OFFSET alternative if keyset pagination is not wired yet.
-- Prefer ListJobs (keyset) for production — offset degrades on large tables.
-- ---------------------------------------------------------------------------
-- name: ListJobsOffset :many
SELECT
    j.id,
    j.title,
    j.company,
    j.location,
    j.salary_min,
    j.salary_max,
    j.created_at
FROM jobs j
ORDER BY j.created_at DESC
LIMIT  $1   -- page_size
OFFSET $2;  -- page_offset


-- ---------------------------------------------------------------------------
-- CountJobs
-- Companion to ListJobsOffset for total-page-count metadata.
-- ---------------------------------------------------------------------------
-- name: CountJobs :one
SELECT COUNT(*) FROM jobs;


-- ---------------------------------------------------------------------------
-- SearchJobs
-- Full-text search on title using the GIN index defined in the migration.
-- Useful extension point — not in the original README but common in job APIs.
-- ---------------------------------------------------------------------------
-- name: SearchJobs :many
SELECT
    j.id,
    j.title,
    j.company,
    j.location,
    j.salary_min,
    j.salary_max,
    j.created_at,
    ts_rank(to_tsvector('english', j.title), plainto_tsquery('english', $1)) AS rank
FROM jobs j
WHERE to_tsvector('english', j.title) @@ plainto_tsquery('english', $1)
ORDER BY rank DESC
LIMIT $2;


-- ---------------------------------------------------------------------------
-- UpdateJob
-- Called by: JobService.UpdateJob  (PATCH /jobs/:id — AdminRole only)
-- Only non-null arguments update their respective columns (COALESCE pattern).
-- ---------------------------------------------------------------------------
-- name: UpdateJob :one
UPDATE jobs
SET
    title       = COALESCE($2, title),
    description = COALESCE($3, description),
    company     = COALESCE($4, company),
    location    = COALESCE($5, location),
    salary_min  = COALESCE($6, salary_min),
    salary_max  = COALESCE($7, salary_max),
    updated_at  = NOW()
WHERE id = $1
RETURNING
    id,
    title,
    description,
    company,
    location,
    salary_min,
    salary_max,
    created_by,
    updated_at;


-- ---------------------------------------------------------------------------
-- DeleteJob
-- Called by: JobService.DeleteJob  (DELETE /jobs/:id — AdminRole only)
-- Hard delete; add a deleted_at column + soft-delete variant if audit trails
-- are required.
-- ---------------------------------------------------------------------------
-- name: DeleteJob :exec
DELETE FROM jobs
WHERE id = $1;
