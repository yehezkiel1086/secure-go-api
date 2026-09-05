-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password_hash,
    role
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING
    id,
    name,
    email,
    role,
    is_email_verified,
    created_at,
    updated_at;

-- name: GetUserByEmail :one
SELECT
    id,
    name,
    email,
    password_hash,
    role,
    is_email_verified,
    email_verify_token_hash,
    email_verify_token_expires_at,
    password_reset_token_hash,
    password_reset_token_expires_at,
    created_at,
    updated_at
FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT
    id,
    name,
    email,
    role,
    is_email_verified,
    created_at,
    updated_at
FROM users
WHERE id = $1
LIMIT 1;

-- name: ListUsers :many
SELECT
    id,
    name,
    email,
    role,
    is_email_verified,
    created_at,
    updated_at
FROM users
ORDER BY created_at DESC
LIMIT  $1
OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: UpdateUserName :one
UPDATE users
SET
    name       = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING
    id,
    name,
    email,
    role,
    is_email_verified,
    updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash                   = $2,
    password_reset_token_hash       = NULL,
    password_reset_token_expires_at = NULL,
    updated_at                      = NOW()
WHERE id = $1;

-- name: SetEmailVerifyToken :exec
UPDATE users
SET
    email_verify_token_hash       = $2,
    email_verify_token_expires_at = $3,
    updated_at                    = NOW()
WHERE id = $1;

-- name: GetUserByEmailVerifyToken :one
SELECT
    id,
    name,
    email,
    role,
    is_email_verified,
    email_verify_token_hash,
    email_verify_token_expires_at,
    updated_at
FROM users
WHERE email_verify_token_hash = $1
LIMIT 1;

-- name: MarkEmailVerified :exec
UPDATE users
SET
    is_email_verified             = TRUE,
    email_verify_token_hash       = NULL,
    email_verify_token_expires_at = NULL,
    updated_at                    = NOW()
WHERE id = $1;

-- name: SetPasswordResetToken :exec
UPDATE users
SET
    password_reset_token_hash       = $2,
    password_reset_token_expires_at = $3,
    updated_at                      = NOW()
WHERE email = $1;

-- name: GetUserByPasswordResetToken :one
SELECT
    id,
    email,
    password_reset_token_hash,
    password_reset_token_expires_at
FROM users
WHERE password_reset_token_hash = $1
LIMIT 1;
