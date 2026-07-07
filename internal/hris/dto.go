package hris

// DashboardResponse is the payload returned by GET /api/hris/dashboard.
type DashboardResponse struct {
CompanyID         int64                 `json:"companyId"`
KPI               KPI                   `json:"kpi"`
SCIAMonthly       []SCIAMonthlyPt       `json:"sciaMonthly"`
AttendanceMonthly []AttendanceMonthlyPt `json:"attendanceMonthly"`
}

// KPI holds the aggregate numbers for the 6 top tiles.
type KPI struct {
TotalEmployees        int     `json:"totalEmployees"`        // active employees
AttendanceRateYtd     float64 `json:"attendanceRateYtd"`     // avg rate, current year months
AttendanceRatePrevYtd float64 `json:"attendanceRatePrevYtd"` // same months, prev year
TotalSakit            int     `json:"totalSakit"`
TotalCuti             int     `json:"totalCuti"`
TotalIzin             int     `json:"totalIzin"`
TotalAlfa             int     `json:"totalAlfa"`
TotalTelat            int     `json:"totalTelat"` // YTD late-arrival count
}

// SCIAMonthlyPt holds approved leave counts per category for one month.
type SCIAMonthlyPt struct {
Month string `json:"month"` // YYYY-MM
Sakit int    `json:"sakit"`
Cuti  int    `json:"cuti"`
Izin  int    `json:"izin"`
Alfa  int    `json:"alfa"`
}

// AttendanceMonthlyPt holds computed attendance rate + late count for one month.
type AttendanceMonthlyPt struct {
Month     string  `json:"month"`     // YYYY-MM
Rate      float64 `json:"rate"`      // %
LateCount int     `json:"lateCount"` // count of late check-ins
}
