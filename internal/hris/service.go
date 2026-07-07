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

func (s *Service) Dashboard(ctx context.Context, companyID int64, year int) (*DashboardResponse, error) {
totalActive, err := s.repo.ActiveEmployeeCount(ctx, companyID)
if err != nil {
return nil, fmt.Errorf("active count: %w", err)
}

attRows, err := s.repo.AttendanceByYear(ctx, companyID, year)
if err != nil {
return nil, fmt.Errorf("attendance: %w", err)
}

leaveRows, err := s.repo.LeaveByYear(ctx, companyID, year)
if err != nil {
return nil, fmt.Errorf("leave: %w", err)
}

// Prev-year data for YTD comparison.
prevAttRows, err := s.repo.AttendanceByYear(ctx, companyID, year-1)
if err != nil {
return nil, fmt.Errorf("prev attendance: %w", err)
}

monthsList := buildMonthsList(year)

// -- Attendance monthly --------------------------------------------------
attMap := make(map[string]attendanceRow, len(attRows))
for _, row := range attRows {
attMap[monthKey(row.Month)] = row
}

attendanceMonthly := make([]AttendanceMonthlyPt, 0, len(monthsList))
var totalTelat int
for _, m := range monthsList {
key := monthKey(m)
row, ok := attMap[key]
rate := computeAttRate(row, m, ok)
lc := 0
if ok {
lc = row.LateCount
totalTelat += lc
}
attendanceMonthly = append(attendanceMonthly, AttendanceMonthlyPt{
Month:     key,
Rate:      rate,
LateCount: lc,
})
}

ytdRate := avgRates(attendanceMonthly)
prevYtdRate := prevYearAvgRate(prevAttRows, monthsList)

// -- SCIA monthly --------------------------------------------------------
leaveMap := make(map[string]map[string]int)
for _, lr := range leaveRows {
key := monthKey(lr.Month)
if leaveMap[key] == nil {
leaveMap[key] = make(map[string]int)
}
leaveMap[key][lr.LeaveCategory] += lr.CaseCount
}

sciaMonthly := make([]SCIAMonthlyPt, 0, len(monthsList))
var totalSakit, totalCuti, totalIzin, totalAlfa int
for _, m := range monthsList {
key := monthKey(m)
cats := leaveMap[key]
sk, cu, iz, al := cats["sakit"], cats["cuti"], cats["izin"], cats["alfa"]
totalSakit += sk
totalCuti += cu
totalIzin += iz
totalAlfa += al
sciaMonthly = append(sciaMonthly, SCIAMonthlyPt{
Month: key, Sakit: sk, Cuti: cu, Izin: iz, Alfa: al,
})
}

return &DashboardResponse{
CompanyID: companyID,
KPI: KPI{
TotalEmployees:        totalActive,
AttendanceRateYtd:     ytdRate,
AttendanceRatePrevYtd: prevYtdRate,
TotalSakit:            totalSakit,
TotalCuti:             totalCuti,
TotalIzin:             totalIzin,
TotalAlfa:             totalAlfa,
TotalTelat:            totalTelat,
},
SCIAMonthly:       sciaMonthly,
AttendanceMonthly: attendanceMonthly,
}, nil
}

// -- helpers -----------------------------------------------------------------

func buildMonthsList(year int) []time.Time {
now := time.Now().UTC()
end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
if year < now.Year() {
end = time.Date(year, 12, 1, 0, 0, 0, 0, time.UTC)
}
start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
var list []time.Time
for cur := start; !cur.After(end); cur = cur.AddDate(0, 1, 0) {
list = append(list, cur)
}
return list
}

func computeAttRate(row attendanceRow, m time.Time, ok bool) float64 {
if !ok || row.DistinctEmployees == 0 {
return 0
}
wd := workingDaysInMonth(m)
expected := float64(row.DistinctEmployees) * float64(wd)
if expected == 0 {
return 0
}
rate := float64(row.AttendanceCount) / expected * 100
if rate > 100 {
rate = 100
}
return roundTo(rate, 1)
}

func avgRates(pts []AttendanceMonthlyPt) float64 {
var sum float64
var n int
for _, p := range pts {
if p.Rate > 0 {
sum += p.Rate
n++
}
}
if n == 0 {
return 0
}
return roundTo(sum/float64(n), 1)
}

func prevYearAvgRate(rows []attendanceRow, currentMonths []time.Time) float64 {
attMap := make(map[string]attendanceRow, len(rows))
for _, r := range rows {
attMap[monthKey(r.Month)] = r
}
var sum float64
var n int
for _, m := range currentMonths {
prevM := time.Date(m.Year()-1, m.Month(), 1, 0, 0, 0, 0, time.UTC)
row, ok := attMap[monthKey(prevM)]
rate := computeAttRate(row, prevM, ok)
if rate > 0 {
sum += rate
n++
}
}
if n == 0 {
return 0
}
return roundTo(sum/float64(n), 1)
}

func monthKey(t time.Time) string { return t.UTC().Format("2006-01") }

// workingDaysInMonth returns a rough estimate of weekdays in the calendar month of t.
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
