-- =============================================================================
--  0014 — Finance dashboard v2
--  Spec MVP: Penjualan Bersih (income), Potongan (40200010), Retur
--  (40300% + 40400%), Biaya Penjualan / Cost of Revenue (50100%).
--  Date filter must support custom range → switch the aggregate from
--  monthly to daily so any [start, end] window is exact.
-- =============================================================================

-- Drop dependent objects first.
DROP VIEW IF EXISTS finance.v_dashboard_monthly;
DROP VIEW IF EXISTS finance.v_account_classified;
DROP TABLE IF EXISTS finance.account_monthly_balance;

-- Per company × day × account, debit/credit sum of POSTED moves.
-- Worker rebuilds last 24 months on every tick.
CREATE TABLE IF NOT EXISTS finance.account_daily_balance (
    company_source_id   INTEGER       NOT NULL,
    date                DATE          NOT NULL,
    account_source_id   INTEGER       NOT NULL,
    debit               NUMERIC(20,2) NOT NULL DEFAULT 0,
    credit              NUMERIC(20,2) NOT NULL DEFAULT 0,
    PRIMARY KEY (company_source_id, date, account_source_id)
);

CREATE INDEX IF NOT EXISTS idx_adb_company_date ON finance.account_daily_balance (company_source_id, date);
CREATE INDEX IF NOT EXISTS idx_adb_account      ON finance.account_daily_balance (account_source_id);

-- Classifier view. Extends previous: adds `cost_of_revenue` (50100%) and
-- merges 40400% into `sales_return` (per spec).
CREATE OR REPLACE VIEW finance.v_account_classified AS
SELECT
    a.*,
    CASE
        WHEN a.code LIKE '40200%'                                       THEN 'sales_discount'
        WHEN a.code LIKE '40300%' OR a.code LIKE '40400%'               THEN 'sales_return'
        WHEN a.internal_group = 'income'  AND a.code LIKE '4%'          THEN 'revenue_operating'
        WHEN a.internal_group = 'income'  AND a.code LIKE '7%'          THEN 'revenue_non_operating'
        WHEN a.internal_group = 'expense' AND a.code LIKE '50100%'      THEN 'cost_of_revenue'
        WHEN a.internal_group = 'expense' AND a.code LIKE '6%'          THEN 'expense_operating'
        WHEN a.internal_group = 'expense' AND a.code LIKE '9%'          THEN 'expense_non_operating'
        ELSE 'other'
    END AS category
FROM finance.accounts a;
