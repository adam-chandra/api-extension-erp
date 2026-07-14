DROP INDEX IF EXISTS finance.idx_accounts_account_type;
ALTER TABLE finance.accounts DROP COLUMN IF EXISTS account_type;
