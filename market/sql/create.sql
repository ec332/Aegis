-- Create market database
CREATE DATABASE IF NOT EXISTS aegis_market;

-- Switch to market database
\c aegis_market;

-- Create markets table
CREATE TABLE IF NOT EXISTS markets (
    id UUID PRIMARY KEY,
    question TEXT NOT NULL,
    description TEXT,
    category VARCHAR(100),
    end_time TIMESTAMP NOT NULL,
    resolution_time TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    outcome VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create options table
CREATE TABLE IF NOT EXISTS options (
    id UUID PRIMARY KEY,
    market_id UUID NOT NULL,
    option_text TEXT NOT NULL,
    current_price DECIMAL(10, 2) NOT NULL DEFAULT 50.00,
    volume DECIMAL(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (market_id) REFERENCES markets(id)
);

-- Create liquidity_pools table
CREATE TABLE IF NOT EXISTS liquidity_pools (
    id UUID PRIMARY KEY,
    market_id UUID NOT NULL,
    total_liquidity DECIMAL(20, 8) NOT NULL DEFAULT 0,
    fee_rate DECIMAL(5, 4) NOT NULL DEFAULT 0.002,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (market_id) REFERENCES markets(id)
);

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    wallet_address VARCHAR(255) NOT NULL UNIQUE,
    balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    nonce TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    market_id UUID NOT NULL,
    option_id UUID NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (market_id) REFERENCES markets(id),
    FOREIGN KEY (option_id) REFERENCES options(id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_markets_status ON markets(status);
CREATE INDEX IF NOT EXISTS idx_markets_end_time ON markets(end_time);
CREATE INDEX IF NOT EXISTS idx_options_market_id ON options(market_id);
CREATE INDEX IF NOT EXISTS idx_liquidity_pools_market_id ON liquidity_pools(market_id);
CREATE INDEX IF NOT EXISTS idx_users_wallet_address ON users(wallet_address);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_market_id ON transactions(market_id);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp);