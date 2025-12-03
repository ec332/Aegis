ALTER TABLE IF EXISTS wallet_accounts ALTER COLUMN id TYPE uuid USING id::uuid;
ALTER TABLE IF EXISTS wallet_accounts ALTER COLUMN user_id TYPE uuid USING user_id::uuid;
DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'wallet_accounts_user_id_fkey') THEN
ALTER TABLE wallet_accounts ADD CONSTRAINT wallet_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);
END IF;
END $$;
