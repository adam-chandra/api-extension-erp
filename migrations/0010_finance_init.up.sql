-- =============================================================================
--  0010 — Finance schema + tables
--  Worker populates these from Odoo account_account / account_move / account_move_line.
--  Strategy: small accounts table (incremental) + two pre-aggregated tables
--  (full-refresh each tick) so BE dashboards stay sub-100ms.
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS finance;

-- Chart of accounts mirrored from Odoo.
CREATE TABLE IF NOT EXISTS finance.accounts (
    id                  BIGSERIAL   PRIMARY KEY,
    source_id           INTEGER     NOT NULL UNIQUE,
    code                TEXT        NOT NULL,
    name                TEXT        NOT NULL,
    internal_group      TEXT,
    company_source_id   INTEGER,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_code     ON finance.accounts (code);
CREATE INDEX IF NOT EXISTS idx_accounts_group    ON finance.accounts (internal_group);
CREATE INDEX IF NOT EXISTS idx_accounts_company  ON finance.accounts (company_source_id);

-- Pre-aggregated: per company × month × account, sum debit/credit of POSTED moves.
-- Worker rebuilds last 24 months on every tick.
CREATE TABLE IF NOT EXISTS finance.account_monthly_balance (
    company_source_id   INTEGER     NOT NULL,
    month               DATE        NOT NULL,
    account_source_id   INTEGER     NOT NULL,
    debit               NUMERIC(20,2) NOT NULL DEFAULT 0,
    credit              NUMERIC(20,2) NOT NULL DEFAULT 0,
    PRIMARY KEY (company_source_id, month, account_source_id)
);

CREATE INDEX IF NOT EXISTS idx_amb_company_month ON finance.account_monthly_balance (company_source_id, month);
CREATE INDEX IF NOT EXISTS idx_amb_account       ON finance.account_monthly_balance (account_source_id);

-- Pre-aggregated: top N retur penjualan lines per company (last 12 months).
-- Worker rebuilds on every tick.
CREATE TABLE IF NOT EXISTS finance.retur_lines (
    line_source_id      INTEGER     PRIMARY KEY,
    move_source_id      INTEGER     NOT NULL,
    account_source_id   INTEGER     NOT NULL,
    company_source_id   INTEGER     NOT NULL,
    date                DATE        NOT NULL,
    description         TEXT,
    amount              NUMERIC(20,2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_retur_company_date ON finance.retur_lines (company_source_id, date DESC);
