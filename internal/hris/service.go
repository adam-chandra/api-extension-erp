package hris

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service { return &Service{repo: r} }

func (s *Service) Dashboard(ctx context.Context, companyID int64) (*DashboardResponse, error) {
	const monthsBack = 12

	totalActive, err := s.repo.ActiveEmployeeCount(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("active count: %w", err)
	}

	deptHC, err := s.repo.DeptHeadcounts(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("dept headcount: %w", err)
	}

	hires, err := s.repo.HiresByMonth(ctx, companyID, monthsBack)
	if err != nil {
		return nil, fmt.Errorf("hires: %w", err)
	}
	exits, err := s.repo.ExitsByMonth(ctx, companyID, monthsBack)
	if err != nil {
		return nil, fmt.Errorf("exits: %w", err)
	}
	att, err := s.repo.AttendanceMonthly(ctx, companyID, monthsBack)
	if err != nil {
		return nil, fmt.Errorf("attendance: %w", err)
	}
	salaries, err := s.repo.SalaryByDept(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("salary by dept: %w", err)
	}

	// Build a chronologically-aligned set of months (last 12 incl. current).
	now := time.Now().UTC()
	startMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -monthsBack+1, 0)
	monthsList := make([]time.Time, 0, monthsBack)
	for cur := startMonth; !cur.After(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)); cur = cur.AddDate(0, 1, 0) {
		monthsList = append(monthsList, cur)
	}

	hireMap := byMonth(hires)
	exitMap := byMonth(exits)
	attMap := attendanceByMonth(att)

	// Workforce trend: hires/exits each month + running headcount tail-anchored
	// to the current active total. We walk backwards: cur_month employees =
	// next_month employees - hires_next + exits_next.
	workforceTrend := make([]WorkforceTrendPt, len(monthsList))
	for i, m := range monthsList {
		key := monthKey(m)
		workforceTrend[i] = WorkforceTrendPt{
			Month: key,
			Hires: hireMap[key],
			Exits: exitMap[key],
		}
	}
	// Anchor the latest month to totalActive, then walk backwards.
	if len(workforceTrend) > 0 {
		workforceTrend[len(workforceTrend)-1].Employees = totalActive
		for i := len(workforceTrend) - 2; i >= 0; i-- {
			next := workforceTrend[i+1]
			workforceTrend[i].Employees = next.Employees - next.Hires + next.Exits
		}
	}

	attendanceTrend := make([]AttendanceTrendPt, len(monthsList))
	for i, m := range monthsList {
		key := monthKey(m)
		row, ok := attMap[key]
		if !ok || row.DistinctEmployees == 0 {
			attendanceTrend[i] = AttendanceTrendPt{Month: key, Rate: 0}
			continue
		}
		// Approximate rate = avg attendance per employee per working day.
		// Working days in a month ≈ 22; max attendance per emp ≈ 22.
		workingDays := workingDaysInMonth(m)
		expected := float64(row.DistinctEmployees) * float64(workingDays)
		if expected == 0 {
			attendanceTrend[i] = AttendanceTrendPt{Month: key, Rate: 0}
			continue
		}
		rate := (float64(row.AttendanceCount) / expected) * 100
		if rate > 100 {
			rate = 100
		}
		attendanceTrend[i] = AttendanceTrendPt{Month: key, Rate: roundTo(rate, 1)}
	}

	// KPI calcs
	var totalHires, totalExits int
	for _, h := range hires {
		totalHires += h.Count
	}
	for _, x := range exits {
		totalExits += x.Count
	}

	// Use last completed month's attendance rate as KPI.
	kpiAttRate := 0.0
	for i := len(attendanceTrend) - 1; i >= 0; i-- {
		if attendanceTrend[i].Rate > 0 {
			kpiAttRate = attendanceTrend[i].Rate
			break
		}
	}

	// Turnover = exits / avg headcount (period start + end)/2
	turnover := 0.0
	if len(workforceTrend) > 1 {
		startHC := workforceTrend[0].Employees
		avg := float64(startHC+totalActive) / 2
		if avg > 0 {
			turnover = roundTo(float64(totalExits)/avg*100, 1)
		}
	}

	recruitment := make([]MonthlyCount, len(monthsList))
	for i, m := range monthsList {
		key := monthKey(m)
		recruitment[i] = MonthlyCount{Month: key, Count: hireMap[key]}
	}

	return &DashboardResponse{
		CompanyID: companyID,
		KPI: KPI{
			TotalEmployees: totalActive,
			AttendanceRate: kpiAttRate,
			NewHires:       totalHires,
			TurnoverRate:   turnover,
		},
		DeptHeadcount:   deptHC,
		Recruitment:     recruitment,
		SalaryByDept:    salaries,
		AttendanceTrend: attendanceTrend,
		WorkforceTrend:  workforceTrend,
	}, nil
}

func byMonth(events []monthlyEvent) map[string]int {
	m := make(map[string]int, len(events))
	for _, e := range events {
		m[monthKey(e.Month)] = e.Count
	}
	return m
}

func attendanceByMonth(rows []attendanceRow) map[string]attendanceRow {
	m := make(map[string]attendanceRow, len(rows))
	for _, r := range rows {
		m[monthKey(r.Month)] = r
	}
	return m
}

func monthKey(t time.Time) string { return t.UTC().Format("2006-01") }

// workingDaysInMonth returns a rough estimate of weekdays in the calendar
// month of `t` (22 ± 1). Holidays not subtracted.
func workingDaysInMonth(t time.Time) int {
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	count := 0
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		wd := d.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			count++
		}
	}
	return count
}

func roundTo(v float64, decimals int) float64 {
	mult := 1.0
	for i := 0; i < decimals; i++ {
		mult *= 10
	}
	return float64(int64(v*mult+0.5)) / mult
}
