-- name: CreateUser :one
INSERT INTO users (
    full_name,
    email,
    password_hash
) VALUES (
    $1, $2, $3
) RETURNING id, full_name, email, password_hash, role, status, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, full_name, email, password_hash, role, status FROM users WHERE LOWER(email) = LOWER($1) LIMIT 1;

-- name: UpdateUserStatus :exec
UPDATE users
SET status = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GetUserByID :one
SELECT id, full_name, email, password_hash, role, status, created_at, updated_at
FROM users
WHERE id = $1 LIMIT 1;
