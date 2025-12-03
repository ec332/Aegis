-- Transaction Service Database Schema

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    market_id UUID NOT NULL,
    option_id UUID NOT NULL,
    transaction_type VARCHAR(10) NOT NULL,
    number_of_shares DECIMAL(20,8) NOT NULL,
    price_per_share DECIMAL(20,8) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_market_id ON transactions(market_id);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_market_id ON transactions(market_id);
CREATE INDEX IF NOT EXISTS idx_transactions_option_id ON transactions(option_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
