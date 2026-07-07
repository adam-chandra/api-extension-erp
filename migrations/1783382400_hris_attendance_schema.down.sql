-- =============================================================================
--  1783382400 — HRIS schema: revert HC Attendance Dashboard migration
-- =============================================================================

-- 3. Drop leave_monthly
DROP TABLE IF EXISTS hris.leave_monthly;

-- 2. Remove late_count from attendance_monthly
ALTER TABLE hris.attendance_monthly
    DROP COLUMN IF EXISTS late_count;

-- 1. Restore salary_by_dept
CREATE TABLE IF NOT EXISTS hris.salary_by_dept (
    company_source_id      INTEGER       NOT NULL,
    department_source_id   INTEGER       NOT NULL,
    month                  DATE          NOT NULL,
    total_salary           NUMERIC(20,2) NOT NULL DEFAULT 0,
    employee_count         INTEGER       NOT NULL DEFAULT 0,
    PRIMARY KEY (company_source_id, department_source_id, month)
);

CREATE INDEX IF NOT EXISTS idx_hris_sal_company_month
    ON hris.salary_by_dept (company_source_id, month);
