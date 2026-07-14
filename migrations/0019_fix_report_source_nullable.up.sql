-- =============================================================================
--  0019 — Fix financial_report_lines: allow NULL report_source_id
--
--  account_financial_report top-level rows have account_report_id = NULL
--  (they are the root nodes of the hierarchy). The NOT NULL constraint was
--  too strict and must be relaxed so the worker can upsert those rows.
-- =============================================================================

ALTER TABLE finance.financial_report_lines
    ALTER COLUMN report_source_id DROP NOT NULL;
