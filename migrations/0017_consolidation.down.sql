-- =============================================================================
--  0017 — Rollback consolidation tables
-- =============================================================================

DROP VIEW IF EXISTS finance.v_consolidated_balance;
DROP VIEW IF EXISTS finance.v_elimination_entries;

DROP TABLE IF EXISTS finance.move_lines;
DROP TABLE IF EXISTS finance.financial_report_lines;
DROP TABLE IF EXISTS finance.financial_reports;
