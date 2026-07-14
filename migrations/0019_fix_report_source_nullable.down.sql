-- =============================================================================
--  0019 — Rollback: restore NOT NULL on report_source_id
-- =============================================================================

ALTER TABLE finance.financial_report_lines
    ALTER COLUMN report_source_id SET NOT NULL;
