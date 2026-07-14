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

// =============================================================================
// Consolidation DTOs
// =============================================================================

// ConsolidationReportRequest is the input for consolidation report queries.
type ConsolidationReportRequest struct {
	CompanyIDs         []int64 `json:"companyIds"`
	StartDate          string  `json:"startDate"`
	EndDate            string  `json:"endDate"`
	ReportType         string  `json:"reportType"`         // "balance_sheet" | "profit_loss"
	IncludeElimination bool    `json:"includeElimination"` // true = sertakan kolom eliminasi
}

type CompanyInfo struct {
	SourceID int64  `json:"sourceId"`
	Name     string `json:"name"`
}

type ConsolidationPeriod struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// ConsolidationReportLine adalah satu baris dalam laporan konsolidasi.
// Dapat berupa:
//   - IsGroup=true          : header section / group (expand/collapse)
//   - IsSubtotal=true       : baris subtotal / total section
//   - IsCarry=true          : subtotal section sebelumnya yang di-carry ke section berikutnya
//   - IsFooter=true         : baris footer laporan (Total Asset, Total Liab & Equity, Selisih)
//   - selain itu            : baris akun detail (leaf)
type ConsolidationReportLine struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Sequence int    `json:"sequence,omitempty"`

	IsGroup    bool `json:"isGroup"`
	IsSubtotal bool `json:"isSubtotal,omitempty"`
	IsCarry    bool `json:"isCarry,omitempty"`  // subtotal section sebelumnya, ditampilkan di awal section baru
	IsFooter   bool `json:"isFooter,omitempty"` // baris footer laporan (Total Asset, dll)

	// Per-company balance: key = companySourceId (string), value = balance
	CompanyBalances map[string]float64 `json:"companyBalances"`

	TotalBeforeElim   float64 `json:"totalBeforeElim"`
	EliminationDebit  float64 `json:"eliminationDebit,omitempty"`
	EliminationCredit float64 `json:"eliminationCredit,omitempty"`
	TotalAfterElim    float64 `json:"totalAfterElim"`

	Children []ConsolidationReportLine `json:"children,omitempty"`
}

type ConsolidationSummary struct {
	TotalBeforeElimination float64 `json:"totalBeforeElimination"`
	TotalEliminationDebit  float64 `json:"totalEliminationDebit"`
	TotalEliminationCredit float64 `json:"totalEliminationCredit"`
	TotalAfterElimination  float64 `json:"totalAfterElimination"`
}

type ConsolidationReportResponse struct {
	ReportType string                    `json:"reportType"`
	Period     ConsolidationPeriod       `json:"period"`
	Companies  []CompanyInfo             `json:"companies"`
	Lines      []ConsolidationReportLine `json:"lines"`
	Summary    ConsolidationSummary      `json:"summary"`
}

// ── Elimination Entries ───────────────────────────────────────────────────────

type EliminationEntriesRequest struct {
	CompanyIDs []int64 `json:"companyIds"`
	StartDate  string  `json:"startDate"`
	EndDate    string  `json:"endDate"`
}

type EliminationEntry struct {
	ID               int64   `json:"id"`
	Date             string  `json:"date"`
	CompanyName      string  `json:"companyName"`
	AccountCode      string  `json:"accountCode"`
	AccountName      string  `json:"accountName"`
	PartnerName      string  `json:"partnerName"`
	IntercompanyWith string  `json:"intercompanyWith"`
	Description      string  `json:"description"`
	Debit            float64 `json:"debit"`
	Credit           float64 `json:"credit"`
}

type EliminationSummary struct {
	TotalDebit  float64 `json:"totalDebit"`
	TotalCredit float64 `json:"totalCredit"`
	EntryCount  int     `json:"entryCount"`
}

type EliminationEntriesResponse struct {
	Period  ConsolidationPeriod `json:"period"`
	Entries []EliminationEntry  `json:"entries"`
	Summary EliminationSummary  `json:"summary"`
}
