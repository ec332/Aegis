ALTER TABLE IF EXISTS users OWNER TO appuser;
ALTER INDEX IF EXISTS idx_users_wallet_address OWNER TO appuser;
ALTER INDEX IF EXISTS idx_users_role OWNER TO appuser;

ALTER TABLE IF EXISTS wallet_accounts OWNER TO appuser;
ALTER INDEX IF EXISTS idx_wallet_accounts_user_id OWNER TO appuser;
ALTER INDEX IF EXISTS idx_wallet_accounts_user_currency OWNER TO appuser;

ALTER TABLE IF EXISTS wallet_transactions OWNER TO appuser;
ALTER INDEX IF EXISTS idx_wallet_transactions_wallet_id OWNER TO appuser;
ALTER INDEX IF EXISTS idx_wallet_transactions_created_at OWNER TO appuser;
ALTER INDEX IF EXISTS idx_wallet_transactions_reference_id OWNER TO appuser;
