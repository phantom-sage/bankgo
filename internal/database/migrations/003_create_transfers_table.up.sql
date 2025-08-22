-- Create transfers table
CREATE TABLE transfers (
    id SERIAL PRIMARY KEY,
    from_account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    to_account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    description TEXT,
    status VARCHAR(20) DEFAULT 'completed' CHECK (status IN ('pending', 'completed', 'failed', 'cancelled')),
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure from and to accounts are different
    CONSTRAINT different_accounts CHECK (from_account_id != to_account_id)
);

-- Create index on from_account_id for transfer history queries (Requirement 3)
CREATE INDEX idx_transfers_from_account ON transfers(from_account_id);

-- Create index on to_account_id for transfer history queries (Requirement 3)
CREATE INDEX idx_transfers_to_account ON transfers(to_account_id);

-- Create index on created_at for sorting transfer history (Requirement 3)
CREATE INDEX idx_transfers_created_at ON transfers(created_at);

-- Create index on status for filtering transfers by status
CREATE INDEX idx_transfers_status ON transfers(status);

-- Create composite index for account transfer history with date ordering
CREATE INDEX idx_transfers_account_date ON transfers(from_account_id, created_at DESC);
CREATE INDEX idx_transfers_to_account_date ON transfers(to_account_id, created_at DESC);