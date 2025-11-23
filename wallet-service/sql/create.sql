-- Create wallet database
CREATE DATABASE IF NOT EXISTS aegis_wallet;

-- Switch to wallet database
\c aegis_wallet;

-- Create wallet_accounts table
CREATE TABLE IF NOT EXISTS wallet_accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    address VARCHAR(255) NOT NULL UNIQUE,
    currency VARCHAR(10) NOT NULL DEFAULT 'USDC',
    total_balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    available_balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create wallet_transactions table
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL,
    market_id UUID,
    type VARCHAR(50) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    reference_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (wallet_id) REFERENCES wallet_accounts(id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_wallet_accounts_user_id ON wallet_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_accounts_address ON wallet_accounts(address);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_type ON wallet_transactions(type);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_status ON wallet_transactions(status);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_created_at ON wallet_transactions(created_at);