-- Create accounts table
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency VARCHAR(3) NOT NULL,
    balance DECIMAL(15,2) DEFAULT 0.00 CHECK (balance >= 0),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Unique constraint: one account per user per currency (Requirement 1)
    CONSTRAINT unique_user_currency UNIQUE(user_id, currency)
);

-- Create index on user_id for faster user account lookups
CREATE INDEX idx_accounts_user_id ON accounts(user_id);

-- Create index on currency for currency-based queries
CREATE INDEX idx_accounts_currency ON accounts(currency);

-- Create index on balance for balance-based queries
CREATE INDEX idx_accounts_balance ON accounts(balance);

-- Create index on created_at for sorting and filtering
CREATE INDEX idx_accounts_created_at ON accounts(created_at);