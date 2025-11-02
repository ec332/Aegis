-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS liquidity_pool CASCADE;
DROP TABLE IF EXISTS options CASCADE;
DROP TABLE IF EXISTS markets CASCADE;

-- Schema will be recreated by the application's InitSchema function
-- Reference schema (matches repository.InitSchema):
/*
CREATE TABLE IF NOT EXISTS markets (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    resolution_datetime TIMESTAMP,
    winning_option_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS options (
    id UUID PRIMARY KEY,
    market_id UUID NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_options_market_id ON options(market_id);

CREATE TABLE IF NOT EXISTS liquidity_pool (
    id UUID PRIMARY KEY,
    market_id UUID NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES options(id) ON DELETE CASCADE,
    pool_value DECIMAL(20, 8) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_liquidity_pool_market_id ON liquidity_pool(market_id);
CREATE INDEX IF NOT EXISTS idx_liquidity_pool_option_id ON liquidity_pool(option_id);
CREATE INDEX IF NOT EXISTS idx_markets_status ON markets(status);
CREATE INDEX IF NOT EXISTS idx_markets_created_at ON markets(created_at);
*/
