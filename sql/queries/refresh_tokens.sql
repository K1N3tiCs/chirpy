-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES(
  $1,
  NOW(),
  NOW(),
  $2,
  NOW() + INTERVAL '60 DAYS',
  NULL
)
RETURNING *;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1
RETURNING *;

-- name: GetUserFromRefreshToken :one
SELECT rt.* FROM refresh_tokens rt
JOIN users u ON u.id = rt.user_id
WHERE rt.token = $1
AND rt.expires_at > NOW()
AND rt.revoked_at IS NULL;
