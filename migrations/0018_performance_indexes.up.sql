-- =============================================================================
--  0018 — Performance indexes for consolidation queries
--
--  account_daily_balance had 51k+ rows causing 1.3-1.5s queries.
--  Covering index lets PG serve the dashboard aggregate entirely from index
--  (no heap access for debit/credit columns).
--  Accounts include index speeds up the view join without extra heap reads.
--  financial_report_lines: add type/domain columns that exist in source.
-- =============================================================================

-- Covering index for dashboard aggregate query:
--   WHERE company_source_id = ? AND date >= ? AND date <= ?
--   GROUP BY date, account_source_id
CREATE INDEX IF NOT EXISTS idx_adb_covering
    ON finance.account_daily_balance (company_source_id, date, account_source_id)
    INCLUDE (debit, credit);

-- Include index on accounts so the JOIN in v_account_classified is index-only
CREATE INDEX IF NOT EXISTS idx_accounts_sid_inc
    ON finance.accounts (source_id)
    INCLUDE (code, internal_group, name);

-- Add missing columns to financial_report_lines to match actual Odoo source.
-- account_financial_report uses 'type' and 'domain' instead of
-- account_codes_operator / account_codes.
ALTER TABLE finance.financial_report_lines
    ADD COLUMN IF NOT EXISTS line_type TEXT,
    ADD COLUMN IF NOT EXISTS domain    TEXT;

-- NOTE: report_source_id nullable fix is handled in 0019 (already applied instances).
