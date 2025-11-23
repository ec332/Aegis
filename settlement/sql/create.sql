-- Settlement Service Database Schema

CREATE TABLE IF NOT EXISTS settlements (
    id VARCHAR(255) PRIMARY KEY,
    market_id VARCHAR(255) NOT NULL,
    winning_option_id VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settlement_distributions (
    id VARCHAR(255) PRIMARY KEY,
    settlement_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (settlement_id) REFERENCES settlements(id)
);

CREATE INDEX IF NOT EXISTS idx_settlements_market_id ON settlements(market_id);
CREATE INDEX IF NOT EXISTS idx_settlements_status ON settlements(status);
CREATE INDEX IF NOT EXISTS idx_settlement_distributions_settlement_id ON settlement_distributions(settlement_id);
CREATE INDEX IF NOT EXISTS idx_settlement_distributions_user_id ON settlement_distributions(user_id);