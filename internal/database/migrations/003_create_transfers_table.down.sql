-- Drop indexes first
DROP INDEX IF EXISTS idx_transfers_to_account_date;
DROP INDEX IF EXISTS idx_transfers_account_date;
DROP INDEX IF EXISTS idx_transfers_status;
DROP INDEX IF EXISTS idx_transfers_created_at;
DROP INDEX IF EXISTS idx_transfers_to_account;
DROP INDEX IF EXISTS idx_transfers_from_account;

-- Drop transfers table
DROP TABLE IF EXISTS transfers;