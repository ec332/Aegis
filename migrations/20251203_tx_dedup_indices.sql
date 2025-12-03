DROP INDEX IF EXISTS idx_transactions_user_id;
DROP INDEX IF EXISTS idx_transactions_market_id;
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_market_id ON transactions(market_id);
CREATE INDEX IF NOT EXISTS idx_transactions_option_id ON transactions(option_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
