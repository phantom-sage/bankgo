-- name: CreateAccount :one
INSERT INTO accounts (
    user_id, currency, balance
) VALUES (
    $1, $2, COALESCE($3, 0.00)
) RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;

-- name: GetAccountForUpdate :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1
FOR UPDATE;

-- name: GetUserAccounts :many
SELECT * FROM accounts
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetAccountByUserAndCurrency :one
SELECT * FROM accounts
WHERE user_id = $1 AND currency = $2 LIMIT 1;

-- name: UpdateAccountBalance :one
UPDATE accounts
SET 
    balance = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts
SET 
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE id = $1 AND balance = 0.00;

-- name: ListAccounts :many
SELECT a.*, u.email, u.first_name, u.last_name
FROM accounts a
JOIN users u ON a.user_id = u.id
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAccountsWithBalance :many
SELECT * FROM accounts
WHERE balance > 0
ORDER BY balance DESC;

-- name: AddToBalance :one
UPDATE accounts
SET 
    balance = balance + $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SubtractFromBalance :one
UPDATE accounts
SET 
    balance = balance - $2,
    updated_at = NOW()
WHERE id = $1 AND balance >= $2
RETURNING *;

-- name: FreezeAccount :one
UPDATE accounts
SET 
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UnfreezeAccount :one
UPDATE accounts
SET 
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetAccountWithUser :one
SELECT a.*, u.email, u.first_name, u.last_name, u.is_active
FROM accounts a
JOIN users u ON a.user_id = u.id
WHERE a.id = $1 LIMIT 1;

-- name: SearchAccounts :many
SELECT a.*, u.email, u.first_name, u.last_name, u.is_active
FROM accounts a
JOIN users u ON a.user_id = u.id
WHERE ($1::text IS NULL OR u.email ILIKE '%' || $1 || '%' OR u.first_name ILIKE '%' || $1 || '%' OR u.last_name ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR a.currency = $2)
  AND ($3::numeric IS NULL OR a.balance >= $3)
  AND ($4::numeric IS NULL OR a.balance <= $4)
  AND ($5::bool IS NULL OR u.is_active = $5)
ORDER BY a.created_at DESC
LIMIT $6 OFFSET $7;

-- name: CountAccounts :one
SELECT COUNT(*)
FROM accounts a
JOIN users u ON a.user_id = u.id
WHERE ($1::text IS NULL OR u.email ILIKE '%' || $1 || '%' OR u.first_name ILIKE '%' || $1 || '%' OR u.last_name ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR a.currency = $2)
  AND ($3::numeric IS NULL OR a.balance >= $3)
  AND ($4::numeric IS NULL OR a.balance <= $4)
  AND ($5::bool IS NULL OR u.is_active = $5);