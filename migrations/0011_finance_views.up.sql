-- =============================================================================
--  0011 — Finance views
--  v_account_classified : adds `category` to every account based on code prefix.
--  v_dashboard_monthly  : per company × month × category, sum of net amount.
--                         `net` follows accounting sign convention:
--                            income  → credit − debit  (positive = real revenue)
--                            expense → debit − credit  (positive = real expense)
-- =============================================================================

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
