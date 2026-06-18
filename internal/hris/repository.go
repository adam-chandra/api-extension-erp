package hris

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	ActiveEmployeeCount(ctx context.Context, companyID int64) (int, error)
	DeptHeadcounts(ctx context.Context, companyID int64) ([]DeptHeadcount, error)
	HiresByMonth(ctx context.Context, companyID int64, monthsBack int) ([]monthlyEvent, error)
	ExitsByMonth(ctx context.Context, companyID int64, monthsBack int) ([]monthlyEvent, error)
	AttendanceMonthly(ctx context.Context, companyID int64, monthsBack int) ([]attendanceRow, error)
	SalaryByDept(ctx context.Context, companyID int64) ([]DeptSalary, error)
}

type monthlyEvent struct {
	Month time.Time `gorm:"column:month"`
	Count int       `gorm:"column:count"`
}

type attendanceRow struct {
	Month             time.Time `gorm:"column:month"`
	WorkedHours       float64   `gorm:"column:worked_hours"`
	AttendanceCount   int       `gorm:"column:attendance_count"`
	DistinctEmployees int       `gorm:"column:distinct_employees"`
}

type gormRepo struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) ActiveEmployeeCount(ctx context.Context, companyID int64) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM hris.employees
		 WHERE company_source_id = ? AND active = TRUE AND departure_date IS NULL`,
		companyID).Scan(&n).Error
	return n, err
}

func (r *gormRepo) DeptHeadcounts(ctx context.Context, companyID int64) ([]DeptHeadcount, error) {
	var rows []DeptHeadcount
	err := r.db.WithContext(ctx).Raw(
		`SELECT
		    COALESCE(d.source_id, 0) AS department_id,
		    COALESCE(d.name, '(Tanpa Departemen)') AS department_name,
		    COUNT(e.*) AS count
		 FROM hris.employees e
		 LEFT JOIN hris.departments d ON d.source_id = e.department_source_id
		 WHERE e.company_source_id = ?
		   AND e.active = TRUE
		   AND e.departure_date IS NULL
		 GROUP BY COALESCE(d.source_id, 0), COALESCE(d.name, '(Tanpa Departemen)')
		 ORDER BY count DESC`,
		companyID).Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) HiresByMonth(ctx context.Context, companyID int64, monthsBack int) ([]monthlyEvent, error) {
	var rows []monthlyEvent
	err := r.db.WithContext(ctx).Raw(
		`SELECT DATE_TRUNC('month', date_of_joining)::date AS month, COUNT(*) AS count
		 FROM hris.employees
		 WHERE company_source_id = ?
		   AND date_of_joining IS NOT NULL
		   AND date_of_joining >= (CURRENT_DATE - make_interval(months => ?))
		 GROUP BY DATE_TRUNC('month', date_of_joining)
		 ORDER BY 1`,
		companyID, monthsBack).Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) ExitsByMonth(ctx context.Context, companyID int64, monthsBack int) ([]monthlyEvent, error) {
	var rows []monthlyEvent
	err := r.db.WithContext(ctx).Raw(
		`SELECT DATE_TRUNC('month', departure_date)::date AS month, COUNT(*) AS count
		 FROM hris.employees
		 WHERE company_source_id = ?
		   AND departure_date IS NOT NULL
		   AND departure_date >= (CURRENT_DATE - make_interval(months => ?))
		 GROUP BY DATE_TRUNC('month', departure_date)
		 ORDER BY 1`,
		companyID, monthsBack).Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) AttendanceMonthly(ctx context.Context, companyID int64, monthsBack int) ([]attendanceRow, error) {
	var rows []attendanceRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT month, worked_hours, attendance_count, distinct_employees
		 FROM hris.attendance_monthly
		 WHERE company_source_id = ?
		   AND month >= (DATE_TRUNC('month', CURRENT_DATE) - make_interval(months => ?))::date
		 ORDER BY month`,
		companyID, monthsBack).Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) SalaryByDept(ctx context.Context, companyID int64) ([]DeptSalary, error) {
	var rows []DeptSalary
	// Use the most recent month available for this company.
	err := r.db.WithContext(ctx).Raw(
		`WITH latest AS (
		     SELECT MAX(month) AS m FROM hris.salary_by_dept WHERE company_source_id = ?
		 )
		 SELECT
		     COALESCE(d.source_id, 0) AS department_id,
		     COALESCE(d.name, '(Tanpa Departemen)') AS department_name,
		     SUM(s.total_salary)::numeric AS total_salary
		 FROM hris.salary_by_dept s
		 LEFT JOIN hris.departments d ON d.source_id = s.department_source_id
		 WHERE s.company_source_id = ?
		   AND s.month = (SELECT m FROM latest)
		 GROUP BY COALESCE(d.source_id, 0), COALESCE(d.name, '(Tanpa Departemen)')
		 ORDER BY total_salary DESC`,
		companyID, companyID).Scan(&rows).Error
	return rows, err
}
