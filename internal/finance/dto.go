package finance

// DashboardResponse is the payload returned by GET /api/finance/dashboard.
type DashboardResponse struct {
	CompanyID int64        `json:"companyId"`
	Period    PeriodInfo   `json:"period"`
	KPI       KPI          `json:"kpi"`
	Trend     []TrendPoint `json:"trend"`
}

type PeriodInfo struct {
	Code  string `json:"code"`  // "all" | "year" | "month" | "custom"
	Start string `json:"start"` // YYYY-MM-DD
	End   string `json:"end"`   // YYYY-MM-DD (inclusive)
}

// KPI numbers used by the 4 top tiles. Values are in IDR.
//
//   - SalesNet       = Σ(credit−debit) for all `income` accounts in the period
//     (naturally nets potongan/retur because they sit on the debit side).
//   - SalesDiscount  = |Σ(debit−credit)| for Potongan Penjualan (40200010).
//   - SalesReturn    = |Σ(debit−credit)| for all Retur accounts (40300%, 40400%).
//   - CostOfRevenue  = Σ(debit−credit) for `expense` accounts coded 50100% (HPP).
type KPI struct {
	SalesNet      float64 `json:"salesNet"`
	SalesDiscount float64 `json:"salesDiscount"`
	SalesReturn   float64 `json:"salesReturn"`
	CostOfRevenue float64 `json:"costOfRevenue"`
}

// TrendPoint is one month bucket in the "Penjualan Bersih vs Biaya Penjualan" chart.
type TrendPoint struct {
	Month         string  `json:"month"` // YYYY-MM
	SalesNet      float64 `json:"salesNet"`
	CostOfRevenue float64 `json:"costOfRevenue"`
}

// ReturLine is one row in the worker-cached top retur lines list.
type ReturLine struct {
	No          int     `json:"no"`
	Description string  `json:"description"`
	Date        string  `json:"date"` // YYYY-MM-DD
	Amount      float64 `json:"amount"`
	Percentage  float64 `json:"percentage"`
}

// ReturAccountRow is one row in the "Retur Penjualan" list view (per-COA aggregate).
type ReturAccountRow struct {
	No         int     `json:"no"`
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Balance    float64 `json:"balance"`    // absolute IDR within the period
	Percentage float64 `json:"percentage"` // balance / sum(balance) * 100
}
