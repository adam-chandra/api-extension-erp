ALTER TABLE finance.accounts
    ADD COLUMN IF NOT EXISTS account_type TEXT;

CREATE INDEX IF NOT EXISTS idx_accounts_account_type ON finance.accounts (account_type);
