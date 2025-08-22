-- Drop indexes first
DROP INDEX IF EXISTS idx_accounts_created_at;
DROP INDEX IF EXISTS idx_accounts_balance;
DROP INDEX IF EXISTS idx_accounts_currency;
DROP INDEX IF EXISTS idx_accounts_user_id;

-- Drop accounts table
DROP TABLE IF EXISTS accounts;