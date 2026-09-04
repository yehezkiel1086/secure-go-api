-- =============================================================================
-- queries/user.sql
-- Maps to: internal/core/port/user.go → UserRepository interface
-- =============================================================================


-- ---------------------------------------------------------------------------
-- CreateUser
-- Called by: UserService.RegisterUser
-- ---------------------------------------------------------------------------
-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password_hash,
    role
)
VALUES (
    $1,  -- name
    $2,  -- email
    $3,  -- password_hash  (bcrypt, never plaintext)
    $4   -- role           ('user' | 'admin')
)
RETURNING
    id,
    name,
    email,
    role,
    is_email_verified,
    created_at,
    updated_at;


-- ---------------------------------------------------------------------------
-- GetUserByEmail
-- Called by: AuthService.Login, AuthService.RequestPasswordReset
-- Note: returns full row including password_hash for credential comparison.
-- ---------------------------------------------------------------------------
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


-- ---------------------------------------------------------------------------
-- GetUserByID
-- Called by: AuthService.RefreshToken (token claims → user lookup)
-- ---------------------------------------------------------------------------
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


-- ---------------------------------------------------------------------------
-- ListUsers
-- Called by: UserService.GetUsers  (Admin-only endpoint: GET /users)
-- Paginated with LIMIT / OFFSET for large result sets.
-- ---------------------------------------------------------------------------
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
LIMIT  $1   -- page_size
OFFSET $2;  -- page_offset


-- ---------------------------------------------------------------------------
-- CountUsers
-- Companion to ListUsers for pagination metadata.
-- ---------------------------------------------------------------------------
-- name: CountUsers :one
SELECT COUNT(*) FROM users;


-- ---------------------------------------------------------------------------
-- UpdateUserName
-- Called by: UserService.UpdateUser (profile edits)
-- ---------------------------------------------------------------------------
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


-- ---------------------------------------------------------------------------
-- UpdateUserPassword
-- Called by: AuthService.ResetPassword (after token validation)
-- Clears the reset token atomically in the same statement.
-- ---------------------------------------------------------------------------
-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash                   = $2,   -- new bcrypt hash
    password_reset_token_hash       = NULL,
    password_reset_token_expires_at = NULL,
    updated_at                      = NOW()
WHERE id = $1;


-- ---------------------------------------------------------------------------
-- SetEmailVerifyToken
-- Called by: UserService.SendVerificationEmail
-- Stores SHA-256 hash; raw token is emailed to the user only.
-- ---------------------------------------------------------------------------
-- name: SetEmailVerifyToken :exec
UPDATE users
SET
    email_verify_token_hash       = $2,   -- SHA-256(raw_token)
    email_verify_token_expires_at = $3,   -- NOW() + EMAIL_TOKEN_DURATION
    updated_at                    = NOW()
WHERE id = $1;


-- ---------------------------------------------------------------------------
-- GetUserByEmailVerifyToken
-- Called by: UserService.VerifyEmail (GET /confirm-email?token=…)
-- Looks up the hashed token; application layer validates expiry + marks verified.
-- ---------------------------------------------------------------------------
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
WHERE email_verify_token_hash = $1   -- SHA-256(raw_token from query param)
LIMIT 1;


-- ---------------------------------------------------------------------------
-- MarkEmailVerified
-- Called by: UserService.VerifyEmail after token validation passes.
-- Clears token columns atomically.
-- ---------------------------------------------------------------------------
-- name: MarkEmailVerified :exec
UPDATE users
SET
    is_email_verified             = TRUE,
    email_verify_token_hash       = NULL,
    email_verify_token_expires_at = NULL,
    updated_at                    = NOW()
WHERE id = $1;


-- ---------------------------------------------------------------------------
-- SetPasswordResetToken
-- Called by: AuthService.RequestPasswordReset
-- Enumeration-safe: the handler always returns 200 regardless of match.
-- ---------------------------------------------------------------------------
-- name: SetPasswordResetToken :exec
UPDATE users
SET
    password_reset_token_hash       = $2,   -- SHA-256(raw_token)
    password_reset_token_expires_at = $3,   -- NOW() + reset window
    updated_at                      = NOW()
WHERE email = $1;


-- ---------------------------------------------------------------------------
-- GetUserByPasswordResetToken
-- Called by: AuthService.ResetPassword
-- Application layer must still check expires_at before accepting the token.
-- ---------------------------------------------------------------------------
-- name: GetUserByPasswordResetToken :one
SELECT
    id,
    email,
    password_reset_token_hash,
    password_reset_token_expires_at
FROM users
WHERE password_reset_token_hash = $1   -- SHA-256(raw_token from request body)
LIMIT 1;
