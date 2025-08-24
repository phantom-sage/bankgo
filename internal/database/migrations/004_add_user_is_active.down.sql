-- Remove is_active column from users table
DROP INDEX IF EXISTS idx_users_is_active;
ALTER TABLE users DROP COLUMN IF EXISTS is_active;