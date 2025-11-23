-- Create settlement database
CREATE DATABASE IF NOT EXISTS aegis_settlement;

-- Switch to settlement database
\c aegis_settlement;

-- Create settlements table
CREATE TABLE IF NOT EXISTS settlements (
    id UUID PRIMARY KEY,
    market_id UUID NOT NULL,
    winning_option_id UUID NOT NULL,
    total_pool DECIMAL(20, 8) NOT NULL,
    winning_pool DECIMAL(20, 8) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    settled_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create settlement_distributions table
CREATE TABLE IF NOT EXISTS settlement_distributions (
    id UUID PRIMARY KEY,
    settlement_id UUID NOT NULL,
    user_id UUID NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (settlement_id) REFERENCES settlements(id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_settlements_market_id ON settlements(market_id);
CREATE INDEX IF NOT EXISTS idx_settlements_status ON settlements(status);
CREATE INDEX IF NOT EXISTS idx_settlement_distributions_settlement_id ON settlement_distributions(settlement_id);
CREATE INDEX IF NOT EXISTS idx_settlement_distributions_user_id ON settlement_distributions(user_id);
CREATE INDEX IF NOT EXISTS idx_settlement_distributions_status ON settlement_distributions(status);