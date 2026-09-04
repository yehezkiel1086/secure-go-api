-- =============================================================================
-- queries/refresh_token.sql
-- Maps to: internal/core/port/auth.go → AuthRepository interface
-- Implements JWT rotation and ErrTokenReuse detection.
--
-- Token lifecycle:
--   Issue  → CreateRefreshToken         (on login / successful refresh)
--   Rotate → GetRefreshToken +
--             RevokeRefreshToken +
--             CreateRefreshToken        (on POST /refresh)
--   Reuse  → RevokeAllUserRefreshTokens (on ErrTokenReuse — full session wipe)
--   Logout → RevokeAllUserRefreshTokens
-- =============================================================================


-- ---------------------------------------------------------------------------
-- CreateRefreshToken
-- Called after successful login or token rotation.
-- Raw token NEVER stored — only SHA-256(raw_token) persists.
-- ---------------------------------------------------------------------------
-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
)
VALUES (
    $1,  -- user_id
    $2,  -- SHA-256(raw_token)
    $3   -- NOW() + REFRESH_TOKEN_DURATION
)
RETURNING
    id,
    user_id,
    token_hash,
    is_revoked,
    expires_at,
    created_at;


-- ---------------------------------------------------------------------------
-- GetRefreshToken
-- Called on POST /refresh to locate the record before rotation.
-- Application layer checks: is_revoked, expires_at — raises ErrTokenReuse
-- if is_revoked = TRUE.
-- ---------------------------------------------------------------------------
-- name: GetRefreshToken :one
SELECT
    id,
    user_id,
    token_hash,
    is_revoked,
    expires_at,
    created_at
FROM refresh_tokens
WHERE token_hash = $1   -- SHA-256(raw_token from HttpOnly cookie)
LIMIT 1;


-- ---------------------------------------------------------------------------
-- RevokeRefreshToken
-- Called during normal token rotation to invalidate the consumed token.
-- The old row is revoked; CreateRefreshToken then issues a fresh row.
-- ---------------------------------------------------------------------------
-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE id = $1;


-- ---------------------------------------------------------------------------
-- RevokeAllUserRefreshTokens
-- Called on:
--   (a) POST /logout  — intentional session termination
--   (b) ErrTokenReuse — a consumed token was replayed; full session wipe
--       signals a potential token theft scenario.
-- ---------------------------------------------------------------------------
-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE user_id  = $1
  AND is_revoked = FALSE;


-- ---------------------------------------------------------------------------
-- DeleteExpiredRefreshTokens
-- Maintenance query — run periodically (e.g. cron / scheduled job) to prune
-- rows that are both expired and revoked, keeping the table lean.
-- ---------------------------------------------------------------------------
-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at  < NOW()
  AND is_revoked  = TRUE;


-- ---------------------------------------------------------------------------
-- CountActiveRefreshTokens
-- Optional: useful for an admin endpoint or health metrics to monitor
-- active session count per user.
-- ---------------------------------------------------------------------------
-- name: CountActiveRefreshTokens :one
SELECT COUNT(*)
FROM refresh_tokens
WHERE user_id   = $1
  AND is_revoked = FALSE
  AND expires_at > NOW();
