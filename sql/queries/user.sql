-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (
    id, email, name, password_hash, role, password_changed_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: UpdateUserRole :one
UPDATE users SET
    role = $2,
    updated_at = $3
WHERE id = $1
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE users SET
    name = $2,
    email = $3,
    updated_at = $4
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users SET
    password_hash = $2,
    password_changed_at = $3,
    updated_at = $3
WHERE id = $1
RETURNING *;
