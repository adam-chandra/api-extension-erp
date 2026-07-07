package hris

import (
"context"
"time"

"gorm.io/gorm"
)

// Repository defines the data access contract for the HRIS attendance dashboard.
type Repository interface {
ActiveEmployeeCount(ctx context.Context, companyID int64) (int, error)
AttendanceByYear(ctx context.Context, companyID int64, year int) ([]attendanceRow, error)
LeaveByYear(ctx context.Context, companyID int64, year int) ([]leaveRow, error)
}

type attendanceRow struct {
Month             time.Time `gorm:"column:month"`
WorkedHours       float64   `gorm:"column:worked_hours"`
AttendanceCount   int       `gorm:"column:attendance_count"`
DistinctEmployees int       `gorm:"column:distinct_employees"`
LateCount         int       `gorm:"column:late_count"`
}

type leaveRow struct {
Month         time.Time `gorm:"column:month"`
LeaveCategory string    `gorm:"column:leave_category"`
CaseCount     int       `gorm:"column:case_count"`
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

func (r *gormRepo) AttendanceByYear(ctx context.Context, companyID int64, year int) ([]attendanceRow, error) {
var rows []attendanceRow
err := r.db.WithContext(ctx).Raw(
`SELECT month, worked_hours, attendance_count, distinct_employees, late_count
 FROM hris.attendance_monthly
 WHERE company_source_id = ?
   AND EXTRACT(year FROM month) = ?
 ORDER BY month`,
companyID, year).Scan(&rows).Error
return rows, err
}

func (r *gormRepo) LeaveByYear(ctx context.Context, companyID int64, year int) ([]leaveRow, error) {
var rows []leaveRow
err := r.db.WithContext(ctx).Raw(
`SELECT month, leave_category, case_count
 FROM hris.leave_monthly
 WHERE company_source_id = ?
   AND EXTRACT(year FROM month) = ?
   AND leave_category != 'other'
 ORDER BY month, leave_category`,
companyID, year).Scan(&rows).Error
return rows, err
}
