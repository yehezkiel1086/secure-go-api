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
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
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
WHERE (j.created_at, j.id) < ($1, $2)
ORDER BY
    j.created_at DESC,
    j.id          DESC
LIMIT $3;

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
LIMIT  $1
OFFSET $2;

-- name: CountJobs :one
SELECT COUNT(*) FROM jobs;

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

-- name: DeleteJob :exec
DELETE FROM jobs
WHERE id = $1;
