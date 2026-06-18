-- Reverse 0014.
DROP VIEW IF EXISTS finance.v_account_classified;
DROP TABLE IF EXISTS finance.account_daily_balance;

CREATE TABLE IF NOT EXISTS finance.account_monthly_balance (
    company_source_id   INTEGER       NOT NULL,
    month               DATE          NOT NULL,
    account_source_id   INTEGER       NOT NULL,
    debit               NUMERIC(20,2) NOT NULL DEFAULT 0,
    credit              NUMERIC(20,2) NOT NULL DEFAULT 0,
    PRIMARY KEY (company_source_id, month, account_source_id)
);

CREATE INDEX IF NOT EXISTS idx_amb_company_month ON finance.account_monthly_balance (company_source_id, month);
CREATE INDEX IF NOT EXISTS idx_amb_account       ON finance.account_monthly_balance (account_source_id);

CREATE OR REPLACE VIEW finance.v_account_classified AS
SELECT
    a.*,
    CASE
        WHEN a.code LIKE '40200%'                                  THEN 'sales_discount'
        WHEN a.code LIKE '40300%'                                  THEN 'sales_return'
        WHEN a.internal_group = 'income'  AND a.code LIKE '4%'     THEN 'revenue_operating'
        WHEN a.internal_group = 'income'  AND a.code LIKE '7%'     THEN 'revenue_non_operating'
        WHEN a.internal_group = 'expense' AND a.code LIKE '6%'     THEN 'expense_operating'
        WHEN a.internal_group = 'expense' AND a.code LIKE '9%'     THEN 'expense_non_operating'
        ELSE 'other'
    END AS category
FROM finance.accounts a;

CREATE OR REPLACE VIEW finance.v_dashboard_monthly AS
SELECT
    b.company_source_id,
    b.month,
    a.category,
    SUM(
        CASE a.internal_group
            WHEN 'income'  THEN b.credit - b.debit
            WHEN 'expense' THEN b.debit  - b.credit
            ELSE                b.debit  - b.credit
        END
    )::NUMERIC(20,2) AS net
FROM finance.account_monthly_balance b
JOIN finance.v_account_classified  a ON a.source_id = b.account_source_id
GROUP BY b.company_source_id, b.month, a.category;
