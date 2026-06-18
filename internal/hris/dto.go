package hris

// DashboardResponse is the payload returned by GET /api/hris/dashboard.
type DashboardResponse struct {
	CompanyID       int64               `json:"companyId"`
	KPI             KPI                 `json:"kpi"`
	DeptHeadcount   []DeptHeadcount     `json:"deptHeadcount"`
	Recruitment     []MonthlyCount      `json:"recruitment"`
	SalaryByDept    []DeptSalary        `json:"salaryByDept"`
	AttendanceTrend []AttendanceTrendPt `json:"attendanceTrend"`
	WorkforceTrend  []WorkforceTrendPt  `json:"workforceTrend"`
}

// KPI numbers used by the 4 top tiles.
type KPI struct {
	TotalEmployees int     `json:"totalEmployees"` // active employees in company
	AttendanceRate float64 `json:"attendanceRate"` // last full month, %
	NewHires       int     `json:"newHires"`       // last 12 months
	TurnoverRate   float64 `json:"turnoverRate"`   // last 12 months: exits/avg headcount, %
}

type DeptHeadcount struct {
	DepartmentID   int64  `json:"departmentId"`
	DepartmentName string `json:"departmentName"`
	Count          int    `json:"count"`
}

type MonthlyCount struct {
	Month string `json:"month"` // YYYY-MM
	Count int    `json:"count"`
}

type DeptSalary struct {
	DepartmentID   int64   `json:"departmentId"`
	DepartmentName string  `json:"departmentName"`
	TotalSalary    float64 `json:"totalSalary"`
}

type AttendanceTrendPt struct {
	Month string  `json:"month"` // YYYY-MM
	Rate  float64 `json:"rate"`  // %
}

type WorkforceTrendPt struct {
	Month     string `json:"month"` // YYYY-MM
	Employees int    `json:"employees"`
	Hires     int    `json:"hires"`
	Exits     int    `json:"exits"`
}
