-- =============================================================================
--  0012 — HRIS schema + tables
--  Source: ethos @ 147.139.196.119 (HRIS Odoo, read-only). Worker populates.
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS hris;

CREATE TABLE IF NOT EXISTS hris.companies (
    source_id           INTEGER     PRIMARY KEY,
    name                TEXT        NOT NULL,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hris.departments (
    source_id           INTEGER     PRIMARY KEY,
    name                TEXT        NOT NULL,
    complete_name       TEXT,
    company_source_id   INTEGER,
    parent_source_id    INTEGER,
    active              BOOLEAN     NOT NULL DEFAULT TRUE,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hris_dept_company ON hris.departments (company_source_id);

CREATE TABLE IF NOT EXISTS hris.employees (
    source_id              INTEGER  PRIMARY KEY,
    name                   TEXT     NOT NULL,
    active                 BOOLEAN  NOT NULL DEFAULT TRUE,
    company_source_id      INTEGER,
    department_source_id   INTEGER,
    job_title              TEXT,
    date_of_joining        DATE,
    departure_date         DATE,
    source_updated_at      TIMESTAMP,
    synced_at              TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hris_emp_company ON hris.employees (company_source_id);
CREATE INDEX IF NOT EXISTS idx_hris_emp_dept    ON hris.employees (department_source_id);
CREATE INDEX IF NOT EXISTS idx_hris_emp_join    ON hris.employees (date_of_joining);
CREATE INDEX IF NOT EXISTS idx_hris_emp_dep     ON hris.employees (departure_date);

-- Per company × month: aggregated attendance signal.
-- worked_hours = total reported worked hours, attendance_count = rows.
-- Worker rebuilds the last 12 months on every tick.
CREATE TABLE IF NOT EXISTS hris.attendance_monthly (
    company_source_id   INTEGER     NOT NULL,
    month               DATE        NOT NULL,
    worked_hours        NUMERIC(20,2) NOT NULL DEFAULT 0,
    attendance_count    INTEGER     NOT NULL DEFAULT 0,
    distinct_employees  INTEGER     NOT NULL DEFAULT 0,
    PRIMARY KEY (company_source_id, month)
);

-- Per company × department × month: payroll total (sum of payslip basic+gross).
-- Sourced from hr_payslip with state='done'. Worker rebuilds last 12 months.
CREATE TABLE IF NOT EXISTS hris.salary_by_dept (
    company_source_id      INTEGER  NOT NULL,
    department_source_id   INTEGER  NOT NULL,
    month                  DATE     NOT NULL,
    total_salary           NUMERIC(20,2) NOT NULL DEFAULT 0,
    employee_count         INTEGER  NOT NULL DEFAULT 0,
    PRIMARY KEY (company_source_id, department_source_id, month)
);

CREATE INDEX IF NOT EXISTS idx_hris_sal_company_month
    ON hris.salary_by_dept (company_source_id, month);
