ALTER TABLE IF EXISTS wallet_transactions ADD COLUMN IF NOT EXISTS reference_id VARCHAR(255);
UPDATE wallet_transactions SET reference_id = description WHERE reference_id IS NULL OR reference_id = '';
ALTER TABLE IF EXISTS wallet_transactions ALTER COLUMN reference_id SET NOT NULL;
ALTER TABLE IF EXISTS wallet_transactions ALTER COLUMN id TYPE uuid USING id::uuid;
ALTER TABLE IF EXISTS wallet_transactions ALTER COLUMN wallet_id TYPE uuid USING wallet_id::uuid;
DROP INDEX IF EXISTS idx_wallet_transactions_reference_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_transactions_reference_id ON wallet_transactions(reference_id);
