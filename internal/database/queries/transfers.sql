-- name: CreateTransfer :one
INSERT INTO transfers (
    from_account_id, to_account_id, amount, description, status
) VALUES (
    $1, $2, $3, COALESCE($4, ''), COALESCE($5, 'completed')
) RETURNING *;

-- name: GetTransfer :one
SELECT t.*, 
       fa.currency as from_currency,
       ta.currency as to_currency,
       fu.email as from_user_email,
       tu.email as to_user_email
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
JOIN users fu ON fa.user_id = fu.id
JOIN users tu ON ta.user_id = tu.id
WHERE t.id = $1 LIMIT 1;

-- name: GetTransfersByAccount :many
SELECT t.*, 
       fa.currency as from_currency,
       ta.currency as to_currency
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
WHERE t.from_account_id = $1 OR t.to_account_id = $1
ORDER BY t.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetTransfersByUser :many
SELECT t.*, 
       fa.currency as from_currency,
       ta.currency as to_currency,
       fa.user_id as from_user_id,
       ta.user_id as to_user_id
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
WHERE fa.user_id = $1 OR ta.user_id = $1
ORDER BY t.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateTransferStatus :one
UPDATE transfers
SET status = $2
WHERE id = $1
RETURNING *;

-- name: ListTransfers :many
SELECT t.*, 
       fa.currency as from_currency,
       ta.currency as to_currency,
       fu.email as from_user_email,
       tu.email as to_user_email
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
JOIN users fu ON fa.user_id = fu.id
JOIN users tu ON ta.user_id = tu.id
ORDER BY t.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetTransfersByStatus :many
SELECT t.*, 
       fa.currency as from_currency,
       ta.currency as to_currency
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
WHERE t.status = $1
ORDER BY t.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetTransfersByDateRange :many
SELECT t.*, 
       fa.currency as from_currency,
       ta.currency as to_currency
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
WHERE t.created_at >= $1 AND t.created_at <= $2
ORDER BY t.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountTransfersByAccount :one
SELECT COUNT(*) FROM transfers
WHERE from_account_id = $1 OR to_account_id = $1;

-- name: SearchTransfersAdvanced :many
SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.description, t.status, t.created_at,
       fa.currency as from_currency, ta.currency as to_currency,
       fu.email as from_user_email, tu.email as to_user_email,
       fu.id as from_user_id, tu.id as to_user_id
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
JOIN users fu ON fa.user_id = fu.id
JOIN users tu ON ta.user_id = tu.id
WHERE ($1::int IS NULL OR fa.user_id = $1 OR ta.user_id = $1)
  AND ($2::int IS NULL OR t.from_account_id = $2 OR t.to_account_id = $2)
  AND ($3::text IS NULL OR fa.currency = $3)
  AND ($4::text IS NULL OR t.status = $4)
  AND ($5::numeric IS NULL OR t.amount >= $5)
  AND ($6::numeric IS NULL OR t.amount <= $6)
  AND ($7::timestamp IS NULL OR t.created_at >= $7)
  AND ($8::timestamp IS NULL OR t.created_at <= $8)
  AND ($9::text IS NULL OR t.description ILIKE '%' || $9 || '%')
ORDER BY t.created_at DESC
LIMIT $10 OFFSET $11;

-- name: CountTransfersAdvanced :one
SELECT COUNT(*)
FROM transfers t
JOIN accounts fa ON t.from_account_id = fa.id
JOIN accounts ta ON t.to_account_id = ta.id
JOIN users fu ON fa.user_id = fu.id
JOIN users tu ON ta.user_id = tu.id
WHERE ($1::int IS NULL OR fa.user_id = $1 OR ta.user_id = $1)
  AND ($2::int IS NULL OR t.from_account_id = $2 OR t.to_account_id = $2)
  AND ($3::text IS NULL OR fa.currency = $3)
  AND ($4::text IS NULL OR t.status = $4)
  AND ($5::numeric IS NULL OR t.amount >= $5)
  AND ($6::numeric IS NULL OR t.amount <= $6)
  AND ($7::timestamp IS NULL OR t.created_at >= $7)
  AND ($8::timestamp IS NULL OR t.created_at <= $8)
  AND ($9::text IS NULL OR t.description ILIKE '%' || $9 || '%');