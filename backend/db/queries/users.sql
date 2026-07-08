-- name: GetUserByEmail :one
SELECT id, email, password_hash, name, global_role, status, last_login_at, email_verified_at, created_at, updated_at
FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, password_hash, name, global_role, status, last_login_at, email_verified_at, created_at, updated_at
FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, name, global_role, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, name, global_role, status, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET name = COALESCE(NULLIF($2, ''), name),
    password_hash = COALESCE(NULLIF($3, ''), password_hash),
    status = COALESCE(NULLIF($4, ''), status),
    global_role = COALESCE(NULLIF($5, ''), global_role),
    last_login_at = COALESCE($6, last_login_at),
    email_verified_at = COALESCE($7, email_verified_at)
WHERE id = $1
RETURNING id, email, name, global_role, status, last_login_at, email_verified_at, created_at, updated_at;

-- name: UpdateLastLogin :exec
UPDATE users
SET last_login_at = $2
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, global_role, status, last_login_at, created_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at;

-- name: GetRefreshToken :one
SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW() LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token_hash = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE user_id = $1;
