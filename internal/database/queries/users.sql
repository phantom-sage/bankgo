-- name: CreateUser :one
INSERT INTO users (
    email, password_hash, first_name, last_name
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET 
    first_name = COALESCE(sqlc.narg(first_name), first_name),
    last_name = COALESCE(sqlc.narg(last_name), last_name),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkWelcomeEmailSent :exec
UPDATE users
SET 
    welcome_email_sent = true,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- Admin-specific user management queries

-- name: AdminListUsers :many
SELECT 
    u.*,
    COUNT(DISTINCT a.id) as account_count,
    COUNT(DISTINCT t.id) as transfer_count
FROM users u
LEFT JOIN accounts a ON u.id = a.user_id
LEFT JOIN transfers t ON (a.id = t.from_account_id OR a.id = t.to_account_id)
WHERE 
    ($3::text IS NULL OR u.email ILIKE '%' || $3 || '%' OR u.first_name ILIKE '%' || $3 || '%' OR u.last_name ILIKE '%' || $3 || '%')
    AND ($4::boolean IS NULL OR u.is_active = $4)
GROUP BY u.id
ORDER BY 
    CASE WHEN $5 = 'email' AND $6 = false THEN u.email END ASC,
    CASE WHEN $5 = 'email' AND $6 = true THEN u.email END DESC,
    CASE WHEN $5 = 'first_name' AND $6 = false THEN u.first_name END ASC,
    CASE WHEN $5 = 'first_name' AND $6 = true THEN u.first_name END DESC,
    CASE WHEN $5 = 'last_name' AND $6 = false THEN u.last_name END ASC,
    CASE WHEN $5 = 'last_name' AND $6 = true THEN u.last_name END DESC,
    CASE WHEN $5 = 'created_at' AND $6 = false THEN u.created_at END ASC,
    CASE WHEN $5 = 'created_at' AND $6 = true THEN u.created_at END DESC,
    u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: AdminCountUsers :one
SELECT COUNT(*)
FROM users u
WHERE 
    ($1::text IS NULL OR u.email ILIKE '%' || $1 || '%' OR u.first_name ILIKE '%' || $1 || '%' OR u.last_name ILIKE '%' || $1 || '%')
    AND ($2::boolean IS NULL OR u.is_active = $2);

-- name: AdminGetUserDetail :one
SELECT 
    u.*,
    COUNT(DISTINCT a.id) as account_count,
    COUNT(DISTINCT t.id) as transfer_count
FROM users u
LEFT JOIN accounts a ON u.id = a.user_id
LEFT JOIN transfers t ON (a.id = t.from_account_id OR a.id = t.to_account_id)
WHERE u.id = $1
GROUP BY u.id;

-- name: AdminCreateUser :one
INSERT INTO users (
    email, password_hash, first_name, last_name, is_active
) VALUES (
    $1, $2, $3, $4, COALESCE($5, true)
) RETURNING *;

-- name: AdminUpdateUser :one
UPDATE users
SET 
    first_name = COALESCE(sqlc.narg(first_name), first_name),
    last_name = COALESCE(sqlc.narg(last_name), last_name),
    is_active = COALESCE(sqlc.narg(is_active), is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AdminDisableUser :exec
UPDATE users
SET 
    is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminEnableUser :exec
UPDATE users
SET 
    is_active = true,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminDeleteUser :exec
DELETE FROM users
WHERE id = $1;