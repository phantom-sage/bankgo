-- Add is_active column to users table for admin disable/enable functionality
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT TRUE;

-- Create index on is_active for filtering
CREATE INDEX idx_users_is_active ON users(is_active);

-- Update existing users to be active by default
UPDATE users SET is_active = TRUE WHERE is_active IS NULL;