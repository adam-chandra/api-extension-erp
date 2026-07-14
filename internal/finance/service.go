package finance

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// ErrInvalidPeriod is returned when the caller asks for an unknown period code.
var ErrInvalidPeriod = errors.New("finance: invalid period")

// ErrInvalidRange is returned when start/end of a custom period don't parse or end < start.
var ErrInvalidRange = errors.New("finance: invalid date range")

const dateFmt = "2006-01-02"

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service { return &Service{repo: r} }

// resolvePeriod converts a short code (and optional custom start/end) into the
// inclusive date window used by every query.
//   - "all":    last 24 months ending today
//   - "year":   1 Jan … 31 Dec of current year
//   - "month":  1st … last day of current month
//   - "custom": parses start, end (YYYY-MM-DD), inclusive
func resolvePeriod(code, customStart, customEnd string, now time.Time) (time.Time, time.Time, error) {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch code {
	case "", "all":
		end := today
		start := end.AddDate(-2, 0, 1)
		return start, end, nil
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
		return start, end, nil
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return start, end, nil
	case "custom":
		if customStart == "" || customEnd == "" {
			return time.Time{}, time.Time{}, ErrInvalidRange
		}
		s, err := time.Parse(dateFmt, customStart)
		if err != nil {
			return time.Time{}, time.Time{}, ErrInvalidRange
		}
		e, err := time.Parse(dateFmt, customEnd)
		if err != nil {
			return time.Time{}, time.Time{}, ErrInvalidRange
		}
		if e.Before(s) {
			return time.Time{}, time.Time{}, ErrInvalidRange
		}
		return s, e, nil
	default:
		return time.Time{}, time.Time{}, ErrInvalidPeriod
	}
}

func (s *Service) Dashboard(ctx context.Context, companyID int64, periodCode, customStart, customEnd string) (*DashboardResponse, error) {
	start, end, err := resolvePeriod(periodCode, customStart, customEnd, time.Now())
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.DailyAggregate(ctx, companyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("daily aggregate: %w", err)
	}

	var kpi KPI
	monthMap := map[string]*TrendPoint{}
	for _, r := range rows {
		switch r.Category {
		// SalesNet = sum of all income net within the period.
		case "revenue_operating", "revenue_non_operating", "sales_discount", "sales_return":
			kpi.SalesNet += r.Net
		case "cost_of_revenue":
			kpi.CostOfRevenue += r.Net
		}
		// Individual tiles (absolute values, more intuitive for a tile).
		switch r.Category {
		case "sales_discount":
			kpi.SalesDiscount += math.Abs(r.Net)
		case "sales_return":
			kpi.SalesReturn += math.Abs(r.Net)
		}

		monthKey := r.Date.Format("2006-01")
		tp, ok := monthMap[monthKey]
		if !ok {
			tp = &TrendPoint{Month: monthKey}
			monthMap[monthKey] = tp
		}
		switch r.Category {
		case "revenue_operating", "revenue_non_operating", "sales_discount", "sales_return":
			tp.SalesNet += r.Net
		case "cost_of_revenue":
			tp.CostOfRevenue += r.Net
		}
	}

	// Build chronologically ordered, gap-free month trend.
	trend := monthRange(start, end, monthMap)

	return &DashboardResponse{
		CompanyID: companyID,
		Period: PeriodInfo{
			Code:  orDefault(periodCode, "all"),
			Start: start.Format(dateFmt),
			End:   end.Format(dateFmt),
		},
		KPI:   kpi,
		Trend: trend,
	}, nil
}

func (s *Service) Returns(ctx context.Context, companyID int64, limit int) ([]ReturLine, error) {
	if limit <= 0 || limit > 200 {
		limit = 15
	}
	rows, err := s.repo.TopReturLines(ctx, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("top retur: %w", err)
	}
	var total float64
	for _, r := range rows {
		total += r.Amount
	}
	out := make([]ReturLine, 0, len(rows))
	for i, r := range rows {
		pct := 0.0
		if total > 0 {
			pct = (r.Amount / total) * 100
		}
		out = append(out, ReturLine{
			No:          i + 1,
			Description: r.Description,
			Date:        r.Date.Format(dateFmt),
			Amount:      r.Amount,
			Percentage:  pct,
		})
	}
	return out, nil
}

// ReturnsByAccount aggregates retur balance per COA within the period.
// Rows with zero balance are dropped (spec: "Jika Returnya 0, tidak perlu
// ditampilkan"). Default limit is 10.
func (s *Service) ReturnsByAccount(ctx context.Context, companyID int64, periodCode, customStart, customEnd string, limit int) ([]ReturAccountRow, error) {
	start, end, err := resolvePeriod(periodCode, customStart, customEnd, time.Now())
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 10
	}

	rows, err := s.repo.ReturBalanceByAccount(ctx, companyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("retur by account: %w", err)
	}

	var total float64
	for _, r := range rows {
		total += r.Balance
	}

	// Sort by balance desc for display, then keep top `limit`.
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Balance > rows[j].Balance })
	if len(rows) > limit {
		rows = rows[:limit]
	}

	out := make([]ReturAccountRow, 0, len(rows))
	for i, r := range rows {
		pct := 0.0
		if total > 0 {
			pct = (r.Balance / total) * 100
		}
		out = append(out, ReturAccountRow{
			No:         i + 1,
			Code:       r.Code,
			Name:       r.Name,
			Balance:    r.Balance,
			Percentage: pct,
		})
	}
	return out, nil
}

// monthRange returns one TrendPoint per month between start and end (inclusive)
// using monthMap when available, zero otherwise.
func monthRange(start, end time.Time, monthMap map[string]*TrendPoint) []TrendPoint {
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	out := make([]TrendPoint, 0)
	for !cur.After(last) {
		key := cur.Format("2006-01")
		if tp, ok := monthMap[key]; ok {
			out = append(out, *tp)
		} else {
			out = append(out, TrendPoint{Month: key})
		}
		cur = cur.AddDate(0, 1, 0)
	}
	return out
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// =============================================================================
// Consolidation — shared building blocks
// =============================================================================

// fallbackAccountType derivasi account_type dari COA prefix.
// Hanya dipakai sebagai safety net untuk akun yang belum ter-sync ulang
// setelah worker fix (account_type masih kosong di finance.accounts).
// Setelah full backfill selesai, fungsi ini tidak akan pernah dipanggil.
func fallbackAccountType(code string) string {
	if len(code) == 0 {
		return ""
	}
	switch string(code[0]) {
	case "1":
		return "asset_current"
	case "2":
		return "liability_current"
	case "3":
		return "equity"
	case "4":
		return "income"
	case "5":
		return "expense_direct_cost"
	case "6":
		return "expense"
	case "7":
		return "income_other"
	case "8":
		return "expense_other"
	case "9":
		return "expense_tax"
	}
	return ""
}

// typeSet builds a lookup set of account_type strings.
func typeSet(types ...string) map[string]bool {
	m := make(map[string]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}

// bucketAccounts holds, for one account_type bucket, every distinct COA code
// (merged across companies) with its per-company balance rows.
type bucketAccounts struct {
	codes  []string
	byCode map[string]map[int64]*consolidatedBalanceRow
	names  map[string]string
}

// collectBucket scans balances and keeps only rows whose (possibly
// fallback-derived) account_type is in wanted, merged by COA code.
func collectBucket(balances []consolidatedBalanceRow, wanted map[string]bool) bucketAccounts {
	byCode := make(map[string]map[int64]*consolidatedBalanceRow)
	names := make(map[string]string)
	for i := range balances {
		b := &balances[i]
		at := b.AccountType
		if at == "" {
			at = fallbackAccountType(b.AccountCode)
		}
		if !wanted[at] {
			continue
		}
		if byCode[b.AccountCode] == nil {
			byCode[b.AccountCode] = make(map[int64]*consolidatedBalanceRow)
		}
		byCode[b.AccountCode][b.CompanySourceID] = b
		names[b.AccountCode] = b.AccountName
	}
	codes := make([]string, 0, len(byCode))
	for c := range byCode {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return bucketAccounts{codes: codes, byCode: byCode, names: names}
}

// buildLeafLines turns one bucket into leaf report lines, one per COA code,
// merged across companies.
func buildLeafLines(b bucketAccounts, includeElim bool, idPrefix string, level int) []ConsolidationReportLine {
	lines := make([]ConsolidationReportLine, 0, len(b.codes))
	for _, code := range b.codes {
		line := ConsolidationReportLine{
			ID:              idPrefix + "-" + code,
			Code:            code,
			Name:            b.names[code],
			Level:           level,
			IsGroup:         false,
			CompanyBalances: make(map[string]float64),
		}
		for compID, row := range b.byCode[code] {
			key := fmt.Sprintf("%d", compID)
			line.CompanyBalances[key] += row.Balance
			line.TotalBeforeElim += row.Balance
			if includeElim {
				line.EliminationDebit += row.EliminationDebit
				line.EliminationCredit += row.EliminationCredit
			}
		}
		line.TotalAfterElim = line.TotalBeforeElim + line.EliminationDebit - line.EliminationCredit
		lines = append(lines, line)
	}
	return lines
}

// buildAggregateLeaf collapses an entire account-type bucket into a single
// leaf row (summed across every COA code and every company), instead of one
// row per COA code. Used for Balance Sheet lines like "Current Asset",
// "Liability", "Equity" that should render as one line, not an expandable
// per-account breakdown.
func buildAggregateLeaf(id, name string, level int, b bucketAccounts, includeElim bool) ConsolidationReportLine {
	line := ConsolidationReportLine{
		ID:              id,
		Name:            name,
		Level:           level,
		IsGroup:         false,
		CompanyBalances: make(map[string]float64),
	}
	for _, code := range b.codes {
		for compID, row := range b.byCode[code] {
			key := fmt.Sprintf("%d", compID)
			line.CompanyBalances[key] += row.Balance
			line.TotalBeforeElim += row.Balance
			if includeElim {
				line.EliminationDebit += row.EliminationDebit
				line.EliminationCredit += row.EliminationCredit
			}
		}
	}
	line.TotalAfterElim = line.TotalBeforeElim + line.EliminationDebit - line.EliminationCredit
	return line
}

// aggregateInto adds child's balances/totals into node (plain sum, sign +1).
func aggregateInto(node *ConsolidationReportLine, child *ConsolidationReportLine) {
	for k, v := range child.CompanyBalances {
		node.CompanyBalances[k] += v
	}
	node.TotalBeforeElim += child.TotalBeforeElim
	node.EliminationDebit += child.EliminationDebit
	node.EliminationCredit += child.EliminationCredit
	node.TotalAfterElim += child.TotalAfterElim
}

// buildGroupNode wraps a set of child lines into a parent group node whose
// own totals are the plain (additive) sum of its children — used for simple
// groupings like Sales Revenue, Operating Expense, CA, NCA, ...
func buildGroupNode(id, name string, level int, children []ConsolidationReportLine) ConsolidationReportLine {
	node := ConsolidationReportLine{
		ID:              id,
		Name:            name,
		Level:           level,
		IsGroup:         true,
		CompanyBalances: make(map[string]float64),
		Children:        children,
	}
	for i := range children {
		aggregateInto(&node, &children[i])
	}
	return node
}

// signedLine is one operand of a combineLines formula: value * sign.
type signedLine struct {
	line *ConsolidationReportLine
	sign float64
}

// combineLines computes a derived row (subtotal / carry / footer) as a
// signed combination of other rows, e.g.
// Total Gross Profit = SalesRevenue(+1) + CostOfRevenue(-1).
func combineLines(id, name string, level int, parts []signedLine) ConsolidationReportLine {
	node := ConsolidationReportLine{
		ID:              id,
		Name:            name,
		Level:           level,
		CompanyBalances: make(map[string]float64),
	}
	for _, p := range parts {
		for k, v := range p.line.CompanyBalances {
			node.CompanyBalances[k] += p.sign * v
		}
		node.TotalBeforeElim += p.sign * p.line.TotalBeforeElim
		node.EliminationDebit += p.sign * p.line.EliminationDebit
		node.EliminationCredit += p.sign * p.line.EliminationCredit
		node.TotalAfterElim += p.sign * p.line.TotalAfterElim
	}
	return node
}

// carryOf duplicates a previously computed row so it can be re-displayed as
// the opening line of the next waterfall section (IsCarry = true).
func carryOf(id string, src ConsolidationReportLine, level int) ConsolidationReportLine {
	c := src
	c.ID = id
	c.Level = level
	c.IsSubtotal = false
	c.IsFooter = false
	c.IsCarry = true
	c.Children = nil
	cb := make(map[string]float64, len(src.CompanyBalances))
	for k, v := range src.CompanyBalances {
		cb[k] = v
	}
	c.CompanyBalances = cb
	return c
}

// sectionTotals sets a section (group) header's own aggregate figures equal
// to its concluding row, so the collapsed section reflects its net result
// rather than a naive (and arithmetically wrong) sum of its children.
func sectionTotals(section *ConsolidationReportLine, final ConsolidationReportLine) {
	section.CompanyBalances = make(map[string]float64, len(final.CompanyBalances))
	for k, v := range final.CompanyBalances {
		section.CompanyBalances[k] = v
	}
	section.TotalBeforeElim = final.TotalBeforeElim
	section.EliminationDebit = final.EliminationDebit
	section.EliminationCredit = final.EliminationCredit
	section.TotalAfterElim = final.TotalAfterElim
}

// =============================================================================
// Profit & Loss — fixed waterfall structure
//
//   Gross Profit             -> Sales Revenue, Cost of Revenue, Total Gross Profit
//   Operating Profit         -> [carry] Total Gross Profit, Operating Expense, Total Operating Profit
//   EBITDA                   -> [carry] Total Operating Profit, Other Income, Other Expense, Total EBITDA
//   EBIT                     -> [carry] Total EBITDA, Amortization, Depreciation, Total EBIT
//   Nett Profit              -> [carry] Total EBIT, Interest, Taxes, Total Nett Profit
//   Comprehensive Income     -> [carry] Total Nett Profit, Total Comprehensive Income
//
// Literal business instruction: "Interest" is sourced from the
// income_interest (INR) bucket. Even though INR is technically income in
// the COA, it is deliberately treated as a deduction at the Nett Profit
// stage and is deliberately excluded from "Other Income".
// =============================================================================

func buildProfitLossLines(balances []consolidatedBalanceRow, includeElim bool) ([]ConsolidationReportLine, ConsolidationSummary) {
	salesRevenueLeaves := buildLeafLines(collectBucket(balances, typeSet("income")), includeElim, "sar", 3)
	costOfRevenueLeaves := buildLeafLines(collectBucket(balances, typeSet("expense_direct_cost")), includeElim, "cor", 3)
	operatingExpenseLeaves := buildLeafLines(collectBucket(balances, typeSet("expense")), includeElim, "ope", 3)
	otherIncomeLeaves := buildLeafLines(collectBucket(balances, typeSet("income_other")), includeElim, "oti", 3)
	otherExpenseLeaves := buildLeafLines(collectBucket(balances, typeSet("expense_other")), includeElim, "ote", 3)
	amortizationLeaves := buildLeafLines(collectBucket(balances, typeSet("expense_amortization")), includeElim, "amo", 3)
	depreciationLeaves := buildLeafLines(collectBucket(balances, typeSet("expense_depreciation")), includeElim, "dep", 3)
	interestLeaves := buildLeafLines(collectBucket(balances, typeSet("income_interest")), includeElim, "inr", 3)
	taxesLeaves := buildLeafLines(collectBucket(balances, typeSet("expense_tax")), includeElim, "tax", 3)

	// ── Gross Profit ─────────────────────────────────────────────────────
	salesRevenue := buildGroupNode("grp-sales-revenue", "Sales Revenue", 2, salesRevenueLeaves)
	costOfRevenue := buildGroupNode("grp-cost-of-revenue", "Cost of Revenue", 2, costOfRevenueLeaves)
	totalGrossProfit := combineLines("sub-total-gross-profit", "Total Gross Profit", 2,
		[]signedLine{{&salesRevenue, 1}, {&costOfRevenue, 1}}) // was -1
	totalGrossProfit.IsSubtotal = true

	grossProfitSection := ConsolidationReportLine{
		ID: "sec-gross-profit", Name: "Gross Profit", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{salesRevenue, costOfRevenue, totalGrossProfit},
	}
	sectionTotals(&grossProfitSection, totalGrossProfit)

	// ── Operating Profit ────────────────────────────────────────────────
	carryGrossProfit := carryOf("carry-gross-profit", totalGrossProfit, 2)
	operatingExpense := buildGroupNode("grp-operating-expense", "Operating Expense", 2, operatingExpenseLeaves)
	totalOperatingProfit := combineLines("sub-total-operating-profit", "Total Operating Profit", 2,
		[]signedLine{{&carryGrossProfit, 1}, {&operatingExpense, 1}}) // was -1
	totalOperatingProfit.IsSubtotal = true

	operatingProfitSection := ConsolidationReportLine{
		ID: "sec-operating-profit", Name: "Operating Profit", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{carryGrossProfit, operatingExpense, totalOperatingProfit},
	}
	sectionTotals(&operatingProfitSection, totalOperatingProfit)

	// ── EBITDA ───────────────────────────────────────────────────────────
	carryOperatingProfit := carryOf("carry-operating-profit", totalOperatingProfit, 2)
	otherIncome := buildGroupNode("grp-other-income", "Other Income", 2, otherIncomeLeaves)
	otherExpense := buildGroupNode("grp-other-expense", "Other Expense", 2, otherExpenseLeaves)
	totalEBITDA := combineLines("sub-total-ebitda", "Total Earnings Before Interest, Taxes, Depreciation, and Amortization", 2,
		[]signedLine{{&carryOperatingProfit, 1}, {&otherIncome, 1}, {&otherExpense, 1}}) // was -1
	totalEBITDA.IsSubtotal = true

	ebitdaSection := ConsolidationReportLine{
		ID: "sec-ebitda", Name: "Earnings Before Interest, Taxes, Depreciation, and Amortization", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{carryOperatingProfit, otherIncome, otherExpense, totalEBITDA},
	}
	sectionTotals(&ebitdaSection, totalEBITDA)

	// ── EBIT ─────────────────────────────────────────────────────────────
	carryEBITDA := carryOf("carry-ebitda", totalEBITDA, 2)
	amortization := buildGroupNode("grp-amortization", "Amortization", 2, amortizationLeaves)
	depreciation := buildGroupNode("grp-depreciation", "Depreciation", 2, depreciationLeaves)
	totalEBIT := combineLines("sub-total-ebit", "Total Earnings Before Interest and Taxes", 2,
		[]signedLine{{&carryEBITDA, 1}, {&amortization, 1}, {&depreciation, 1}})
	totalEBIT.IsSubtotal = true

	ebitSection := ConsolidationReportLine{
		ID: "sec-ebit", Name: "Earnings Before Interest and Taxes", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{carryEBITDA, amortization, depreciation, totalEBIT},
	}
	sectionTotals(&ebitSection, totalEBIT)

	// ── Nett Profit ──────────────────────────────────────────────────────
	carryEBIT := carryOf("carry-ebit", totalEBIT, 2)
	interest := buildGroupNode("grp-interest", "Interest", 2, interestLeaves) // income_interest → naturally positive
	taxes := buildGroupNode("grp-taxes", "Taxes", 2, taxesLeaves)             // expense_tax → naturally positive (debit-credit)
	totalNettProfit := combineLines("sub-total-nett-profit", "Total Nett Profit", 2,
		[]signedLine{{&carryEBIT, 1}, {&interest, 1}, {&taxes, -1}}) // unchanged: both still need explicit subtraction
	totalNettProfit.IsSubtotal = true

	nettProfitSection := ConsolidationReportLine{
		ID: "sec-nett-profit", Name: "Nett Profit", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{carryEBIT, interest, taxes, totalNettProfit},
	}
	sectionTotals(&nettProfitSection, totalNettProfit)

	// ── Comprehensive Income ─────────────────────────────────────────────
	carryNettProfit := carryOf("carry-nett-profit", totalNettProfit, 2)
	totalComprehensiveIncome := combineLines("sub-total-comprehensive-income", "Total Comprehensive Income", 2,
		[]signedLine{{&carryNettProfit, 1}})
	totalComprehensiveIncome.IsSubtotal = true

	comprehensiveIncomeSection := ConsolidationReportLine{
		ID: "sec-comprehensive-income", Name: "Comprehensive Income", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{carryNettProfit, totalComprehensiveIncome},
	}
	sectionTotals(&comprehensiveIncomeSection, totalComprehensiveIncome)

	lines := []ConsolidationReportLine{
		grossProfitSection,
		operatingProfitSection,
		ebitdaSection,
		ebitSection,
		nettProfitSection,
		comprehensiveIncomeSection,
	}

	summary := ConsolidationSummary{
		TotalBeforeElimination: totalComprehensiveIncome.TotalBeforeElim,
		TotalEliminationDebit:  totalComprehensiveIncome.EliminationDebit,
		TotalEliminationCredit: totalComprehensiveIncome.EliminationCredit,
		TotalAfterElimination:  totalComprehensiveIncome.TotalAfterElim,
	}
	return lines, summary
}

// =============================================================================
// Balance Sheet — Assets / Liability / Equity / Balance, each section
// collapsed to single aggregate lines (not per-COA), closed by a subtotal.
// =============================================================================

// buildBalanceSheetLines — CYE section replaced with a derived calculation.
func buildBalanceSheetLines(balances []consolidatedBalanceRow, cyeIncomeExpense []consolidatedBalanceRow, includeElim bool) ([]ConsolidationReportLine, ConsolidationSummary) {
	caLeaves := buildLeafLines(collectBucket(balances, typeSet(
		"asset_receivable", "asset_cash", "asset_current", "asset_prepayments")), includeElim, "ca", 3)
	ncaLeaves := buildLeafLines(collectBucket(balances, typeSet("asset_fixed", "asset_non_current")), includeElim, "nca", 3)
	clLeaves := buildLeafLines(collectBucket(balances, typeSet(
		"liability_payable", "liability_credit_card", "liability_current")), includeElim, "cl", 3)
	nclLeaves := buildLeafLines(collectBucket(balances, typeSet("liability_non_current")), includeElim, "ncl", 3)
	eqLeaves := buildLeafLines(collectBucket(balances, typeSet("equity")), includeElim, "eq", 3)

	// ── Current Year Earnings (derived) ────────────────────────────────
	// Odoo never posts to the CYE account directly during the year — it's
	// a computed figure: Total Income − Total Expense for the fiscal
	// year-to-date. We reuse the same P&L-style aggregation, fed by
	// year-to-date income/expense movement (cyeIncomeExpense), not by
	// searching for an "equity_unaffected" GL balance that will never exist.
	incomeLeaves := buildLeafLines(collectBucket(cyeIncomeExpense, typeSet(
		"income", "income_other", "income_interest")), includeElim, "cye-inc", 3)
	expenseLeaves := buildLeafLines(collectBucket(cyeIncomeExpense, typeSet(
		"expense", "expense_direct_cost", "expense_other",
		"expense_depreciation", "expense_amortization", "expense_tax")), includeElim, "cye-exp", 3)

	incomeGroup := buildGroupNode("grp-cye-income", "Income (YTD)", 3, incomeLeaves)
	expenseGroup := buildGroupNode("grp-cye-expense", "Expense (YTD)", 3, expenseLeaves)

	currentYearEarnings := combineLines("grp-current-year-earnings", "Current Year Earnings", 2,
		[]signedLine{{&incomeGroup, 1}, {&expenseGroup, -1}})

	// ── Assets ───────────────────────────────────────────────────────────
	currentAsset := buildGroupNode("grp-current-asset", "Current Asset", 2, caLeaves)
	nonCurrentAsset := buildGroupNode("grp-non-current-asset", "Non-Current Asset", 2, ncaLeaves)
	totalAssets := combineLines("sub-total-assets", "Total Assets", 2,
		[]signedLine{{&currentAsset, 1}, {&nonCurrentAsset, 1}})
	totalAssets.IsSubtotal = true

	assetsSection := ConsolidationReportLine{
		ID: "sec-assets", Name: "Assets", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{currentAsset, nonCurrentAsset, totalAssets},
	}
	sectionTotals(&assetsSection, totalAssets)

	// ── Liability ────────────────────────────────────────────────────────
	liability := buildGroupNode("grp-liability", "Liability", 2, clLeaves)
	nonCurrentLiabilities := buildGroupNode("grp-non-current-liabilities", "Non Current Liabilities", 2, nclLeaves)
	totalLiability := combineLines("sub-total-liability", "Total Liability", 2,
		[]signedLine{{&liability, 1}, {&nonCurrentLiabilities, 1}})
	totalLiability.IsSubtotal = true

	liabilitySection := ConsolidationReportLine{
		ID: "sec-liability", Name: "Liability", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{liability, nonCurrentLiabilities, totalLiability},
	}
	sectionTotals(&liabilitySection, totalLiability)

	// ── Equity ───────────────────────────────────────────────────────────
	equity := buildGroupNode("grp-equity", "Equity", 2, eqLeaves)
	totalEquity := combineLines("sub-total-equity", "Total Equity", 2,
		[]signedLine{{&equity, 1}, {&currentYearEarnings, 1}})
	totalEquity.IsSubtotal = true

	equitySection := ConsolidationReportLine{
		ID: "sec-equity", Name: "Equity", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{equity, currentYearEarnings, totalEquity},
	}
	sectionTotals(&equitySection, totalEquity)

	// ── Balance ──────────────────────────────────────────────────────────
	carryTotalAssets := carryOf("carry-total-assets", totalAssets, 2)
	totalLiabilitiesAndEquity := combineLines("carry-total-liab-equity", "Total Liabilities and Equity", 2,
		[]signedLine{{&totalLiability, 1}, {&totalEquity, 1}})
	totalLiabilitiesAndEquity.IsCarry = true

	totalBalance := combineLines("sub-total-balance", "Total Balance", 2,
		[]signedLine{{&carryTotalAssets, 1}, {&totalLiabilitiesAndEquity, -1}})
	totalBalance.IsSubtotal = true

	balanceSection := ConsolidationReportLine{
		ID: "sec-balance", Name: "Balance", Level: 1, IsGroup: true,
		CompanyBalances: make(map[string]float64),
		Children:        []ConsolidationReportLine{carryTotalAssets, totalLiabilitiesAndEquity, totalBalance},
	}
	sectionTotals(&balanceSection, totalBalance)

	lines := []ConsolidationReportLine{
		assetsSection,
		liabilitySection,
		equitySection,
		balanceSection,
	}

	summary := ConsolidationSummary{
		TotalBeforeElimination: totalBalance.TotalBeforeElim,
		TotalEliminationDebit:  totalBalance.EliminationDebit,
		TotalEliminationCredit: totalBalance.EliminationCredit,
		TotalAfterElimination:  totalBalance.TotalAfterElim,
	}
	return lines, summary
}

func (s *Service) GetConsolidationReport(ctx context.Context, req ConsolidationReportRequest) (*ConsolidationReportResponse, error) {
	if len(req.CompanyIDs) == 0 {
		return nil, errors.New("at least one company required")
	}

	startDate, err := time.Parse(dateFmt, req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate: %w", err)
	}
	endDate, err := time.Parse(dateFmt, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid endDate: %w", err)
	}

	var coaPrefixes []string
	switch req.ReportType {
	case "profit_loss":
		coaPrefixes = []string{"4", "5", "6", "7", "8", "9"}
	default: // balance_sheet
		coaPrefixes = []string{"1", "2", "3"}
	}

	balances, err := s.repo.GetConsolidatedBalance(ctx, req.CompanyIDs, startDate, endDate, coaPrefixes)
	if err != nil {
		return nil, fmt.Errorf("get consolidated balance: %w", err)
	}

	// Build company map
	companyMap := make(map[int64]CompanyInfo)
	for i := range balances {
		b := &balances[i]
		companyMap[b.CompanySourceID] = CompanyInfo{
			SourceID: b.CompanySourceID,
			Name:     b.CompanyName,
		}
	}
	companies := make([]CompanyInfo, 0, len(companyMap))
	for _, c := range companyMap {
		companies = append(companies, c)
	}
	for _, cid := range req.CompanyIDs {
		if _, ok := companyMap[cid]; !ok {
			companies = append(companies, CompanyInfo{SourceID: cid})
		}
	}
	sort.Slice(companies, func(i, j int) bool {
		if companies[i].Name != companies[j].Name {
			return companies[i].Name < companies[j].Name
		}
		return companies[i].SourceID < companies[j].SourceID
	})

	var lines []ConsolidationReportLine
	var summary ConsolidationSummary
	if req.ReportType == "profit_loss" {
		lines, summary = buildProfitLossLines(balances, req.IncludeElimination)
	} else {
		// Fiscal year start for the requested end date — adjust if your
		// fiscal year doesn't start Jan 1.
		fyStart := time.Date(endDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		cyeMovement, err := s.repo.GetConsolidatedBalance(ctx, req.CompanyIDs, fyStart, endDate, []string{"4", "5", "6", "7", "8", "9"})
		if err != nil {
			return nil, fmt.Errorf("get cye movement: %w", err)
		}
		lines, summary = buildBalanceSheetLines(balances, cyeMovement, req.IncludeElimination)
	}

	return &ConsolidationReportResponse{
		ReportType: req.ReportType,
		Period: ConsolidationPeriod{
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
		},
		Companies: companies,
		Lines:     lines,
		Summary:   summary,
	}, nil
}

func (s *Service) GetEliminationEntries(ctx context.Context, req EliminationEntriesRequest) (*EliminationEntriesResponse, error) {
	if len(req.CompanyIDs) == 0 {
		return nil, errors.New("at least one company required")
	}
	startDate, err := time.Parse(dateFmt, req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate: %w", err)
	}
	endDate, err := time.Parse(dateFmt, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid endDate: %w", err)
	}

	entries, err := s.repo.GetEliminationEntries(ctx, req.CompanyIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get elimination entries: %w", err)
	}

	var summary EliminationSummary
	elimEntries := make([]EliminationEntry, 0, len(entries))
	for _, e := range entries {
		summary.TotalDebit += e.Debit
		summary.TotalCredit += e.Credit
		summary.EntryCount++
		elimEntries = append(elimEntries, EliminationEntry{
			ID:               e.ID,
			Date:             e.Date,
			CompanyName:      e.CompanyName,
			AccountCode:      e.AccountCode,
			AccountName:      e.AccountName,
			PartnerName:      e.PartnerName,
			IntercompanyWith: e.IntercompanyPartnerName,
			Description:      e.Description,
			Debit:            e.Debit,
			Credit:           e.Credit,
		})
	}

	return &EliminationEntriesResponse{
		Period:  ConsolidationPeriod{StartDate: req.StartDate, EndDate: req.EndDate},
		Entries: elimEntries,
		Summary: summary,
	}, nil
}
