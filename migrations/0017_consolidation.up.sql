-- =============================================================================
--  0017 — Consolidation support: financial reports + GL entries
--  
--  Financial reports structure from account.financial.report (Odoo) provides
--  the hierarchy for P&L and Balance Sheet. Move lines table stores detailed
--  GL entries for drill-down, including inter-company transaction flag.
-- =============================================================================

-- Financial report templates (source: account_financial_html_report)
CREATE TABLE IF NOT EXISTS finance.financial_reports (
    id                  BIGSERIAL   PRIMARY KEY,
    source_id           INTEGER     NOT NULL UNIQUE,
    name                TEXT        NOT NULL,
    report_type         TEXT,                       -- 'profit_loss', 'balance_sheet', etc
    is_consolidated     BOOLEAN     NOT NULL DEFAULT FALSE,
    source_created_at   TIMESTAMP,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fin_reports_type ON finance.financial_reports (report_type);

-- Financial report line items with hierarchy (source: account_financial_html_report_line)
CREATE TABLE IF NOT EXISTS finance.financial_report_lines (
    id                  BIGSERIAL   PRIMARY KEY,
    source_id           INTEGER     NOT NULL UNIQUE,
    report_source_id    INTEGER     NOT NULL,       -- FK to financial_reports.source_id
    parent_source_id    INTEGER,                    -- Self-referencing hierarchy
    name                TEXT        NOT NULL,
    code                TEXT,                       -- For matching with accounts
    sequence            INTEGER,
    level               INTEGER,
    formulas            TEXT,                       -- JSON string with calculation rules
    account_codes_operator TEXT,                    -- 'include', 'exclude', etc
    account_codes       TEXT,                       -- Comma-separated account codes
    source_created_at   TIMESTAMP,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fin_lines_report ON finance.financial_report_lines (report_source_id);
CREATE INDEX IF NOT EXISTS idx_fin_lines_parent ON finance.financial_report_lines (parent_source_id);
CREATE INDEX IF NOT EXISTS idx_fin_lines_code   ON finance.financial_report_lines (code);

-- Detailed GL entries for drill-down (source: account_move_line)
-- Stores last 24 months of POSTED entries with inter-company flag.
-- Worker refreshes this table periodically (full or incremental).
CREATE TABLE IF NOT EXISTS finance.move_lines (
    id                      BIGSERIAL       PRIMARY KEY,
    source_id               INTEGER         NOT NULL UNIQUE,
    move_source_id          INTEGER         NOT NULL,       -- account_move.id
    company_source_id       INTEGER         NOT NULL,
    account_source_id       INTEGER         NOT NULL,
    partner_source_id       INTEGER,
    journal_source_id       INTEGER,
    
    date                    DATE            NOT NULL,
    name                    TEXT,                           -- Entry label/description
    ref                     TEXT,                           -- Move reference
    
    debit                   NUMERIC(20,2)   NOT NULL DEFAULT 0,
    credit                  NUMERIC(20,2)   NOT NULL DEFAULT 0,
    balance                 NUMERIC(20,2)   NOT NULL DEFAULT 0,
    amount_currency         NUMERIC(20,2),
    currency_id             INTEGER,
    
    -- Inter-company transaction flag
    is_intercompany         BOOLEAN         NOT NULL DEFAULT FALSE,
    intercompany_partner_id INTEGER,                        -- Related company as partner
    
    -- Move state
    move_state              TEXT            NOT NULL,       -- 'draft', 'posted', 'cancel'
    
    source_created_at       TIMESTAMP,
    source_updated_at       TIMESTAMP,
    synced_at               TIMESTAMP       NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_move_lines_company    ON finance.move_lines (company_source_id);
CREATE INDEX IF NOT EXISTS idx_move_lines_account    ON finance.move_lines (account_source_id);
CREATE INDEX IF NOT EXISTS idx_move_lines_date       ON finance.move_lines (date DESC);
CREATE INDEX IF NOT EXISTS idx_move_lines_move       ON finance.move_lines (move_source_id);
CREATE INDEX IF NOT EXISTS idx_move_lines_partner    ON finance.move_lines (partner_source_id);
CREATE INDEX IF NOT EXISTS idx_move_lines_state      ON finance.move_lines (move_state);
CREATE INDEX IF NOT EXISTS idx_move_lines_interco    ON finance.move_lines (is_intercompany) WHERE is_intercompany = TRUE;

-- View: Consolidation elimination entries
-- Identifies inter-company transactions that should be eliminated in consolidated reports.
CREATE OR REPLACE VIEW finance.v_elimination_entries AS
SELECT
    ml.id,
    ml.source_id,
    ml.company_source_id,
    ml.account_source_id,
    ml.date,
    ml.debit,
    ml.credit,
    ml.balance,
    ml.partner_source_id,
    ml.intercompany_partner_id,
    ml.name,
    ml.ref,
    a.code AS account_code,
    a.name AS account_name,
    c.name AS company_name
FROM finance.move_lines ml
JOIN finance.accounts a ON a.source_id = ml.account_source_id
JOIN auth.companies c ON c.source_id = ml.company_source_id
WHERE ml.is_intercompany = TRUE
  AND ml.move_state = 'posted';

-- View: Consolidated balance per account (with elimination)
-- Sums balances across all companies, then subtracts inter-company eliminations.
CREATE OR REPLACE VIEW finance.v_consolidated_balance AS
WITH total_balance AS (
    SELECT
        account_source_id,
        date,
        SUM(debit) AS total_debit,
        SUM(credit) AS total_credit,
        SUM(balance) AS total_balance
    FROM finance.move_lines
    WHERE move_state = 'posted'
    GROUP BY account_source_id, date
),
elimination_balance AS (
    SELECT
        account_source_id,
        date,
        SUM(debit) AS elim_debit,
        SUM(credit) AS elim_credit,
        SUM(balance) AS elim_balance
    FROM finance.move_lines
    WHERE is_intercompany = TRUE
      AND move_state = 'posted'
    GROUP BY account_source_id, date
)
SELECT
    tb.account_source_id,
    tb.date,
    tb.total_debit,
    tb.total_credit,
    tb.total_balance,
    COALESCE(eb.elim_debit, 0) AS elimination_debit,
    COALESCE(eb.elim_credit, 0) AS elimination_credit,
    COALESCE(eb.elim_balance, 0) AS elimination_balance,
    tb.total_balance - COALESCE(eb.elim_balance, 0) AS consolidated_balance
FROM total_balance tb
LEFT JOIN elimination_balance eb 
    ON eb.account_source_id = tb.account_source_id 
    AND eb.date = tb.date;
