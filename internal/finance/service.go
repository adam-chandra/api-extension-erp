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
		start := end.AddDate(-2, 0, 1) // ~24 months
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
