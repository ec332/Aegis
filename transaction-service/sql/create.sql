
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL,          -- The user who executed the transaction
    market_id UUID NOT NULL,        -- The market/instrument the transaction belongs to
    option_id UUID NOT NULL,        -- The specific option/contract traded

    transaction_type VARCHAR(50) NOT NULL,  -- e.g., 'BUY', 'SELL', 'SPLIT'
    number_of_shares NUMERIC(10, 4) NOT NULL, -- Total shares/quantity traded
    price_per_share NUMERIC(10, 4) NOT NULL,  -- Price per unit at the time of transaction

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() -- Time of creation
);
CREATE INDEX idx_transactions_created_at ON transactions (created_at);
CREATE INDEX idx_transactions_user_id ON transactions (user_id);
