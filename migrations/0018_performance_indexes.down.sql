-- =============================================================================
--  0018 — Rollback performance indexes
-- =============================================================================

DROP INDEX IF EXISTS finance.idx_adb_covering;
DROP INDEX IF EXISTS finance.idx_accounts_sid_inc;

ALTER TABLE finance.financial_report_lines
    DROP COLUMN IF EXISTS line_type,
    DROP COLUMN IF EXISTS domain;

ALTER TABLE finance.financial_report_lines
    ALTER COLUMN report_source_id SET NOT NULL;
