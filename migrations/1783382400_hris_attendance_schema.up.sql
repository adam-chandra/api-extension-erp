-- =============================================================================
--  1783382400 — HRIS schema: migrate to HC Attendance Dashboard
--
--  Changes:
--    1. DROP hris.salary_by_dept  — not needed for attendance dashboard
--    2. ADD late_count to hris.attendance_monthly
--    3. CREATE hris.leave_monthly  — SCIA (Sakit/Cuti/Izin/Alfa) per company × month
-- =============================================================================

-- 1. Drop salary table (replaced by attendance-focused dashboard)
DROP TABLE IF EXISTS hris.salary_by_dept;

-- 2. Add late arrival count to existing attendance aggregate
ALTER TABLE hris.attendance_monthly
    ADD COLUMN IF NOT EXISTS late_count INTEGER NOT NULL DEFAULT 0;

-- 3. SCIA leave counts per company × month × category
--    leave_category: 'sakit' | 'cuti' | 'izin' | 'alfa' | 'other'
--    case_count: number of approved leave requests in that bucket
CREATE TABLE IF NOT EXISTS hris.leave_monthly (
    company_source_id   INTEGER  NOT NULL,
    month               DATE     NOT NULL,
    leave_category      TEXT     NOT NULL,
    case_count          INTEGER  NOT NULL DEFAULT 0,
    PRIMARY KEY (company_source_id, month, leave_category)
);

CREATE INDEX IF NOT EXISTS idx_hris_leave_company_month
    ON hris.leave_monthly (company_source_id, month);
