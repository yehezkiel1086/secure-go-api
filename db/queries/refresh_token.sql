-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING
    id,
    user_id,
    token_hash,
    is_revoked,
    expires_at,
    created_at;

-- name: GetRefreshToken :one
SELECT
    id,
    user_id,
    token_hash,
    is_revoked,
    expires_at,
    created_at
FROM refresh_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE user_id  = $1
  AND is_revoked = FALSE;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at  < NOW()
  AND is_revoked  = TRUE;

-- name: CountActiveRefreshTokens :one
SELECT COUNT(*)
FROM refresh_tokens
WHERE user_id   = $1
  AND is_revoked = FALSE
  AND expires_at > NOW();
