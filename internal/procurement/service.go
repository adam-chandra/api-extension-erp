package procurement

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidPeriod = errors.New("procurement: invalid period")
var ErrInvalidRange = errors.New("procurement: invalid date range")

const dateFmt = "2006-01-02"

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service { return &Service{repo: r} }

// parseCustomRange is a helper to parse and validate custom date range input.
func parseCustomRange(startStr, endStr string) (time.Time, time.Time, error) {
	if startStr == "" || endStr == "" {
		return time.Time{}, time.Time{}, ErrInvalidRange
	}
	start, err := time.Parse(dateFmt, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidRange
	}
	end, err := time.Parse(dateFmt, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidRange
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, ErrInvalidRange
	}
	return start, end, nil
}

func resolvePeriod(code, customStart, customEnd string, now time.Time) (time.Time, time.Time, error) {
	now = now.UTC()
	switch code {
	case "", "all":
		// All Time covers all historical and future transactions
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC), nil
	case "year":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC), nil
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, -1), nil
	case "custom":
		return parseCustomRange(customStart, customEnd)
	default:
		return time.Time{}, time.Time{}, ErrInvalidPeriod
	}
}

func resolvePOCycleTimeTrendPeriod(code, customStart, customEnd string, now time.Time) (time.Time, time.Time, error) {
	now = now.UTC()
	switch code {
	case "", "all", "month":
		// Bulan berjalan s/d 11 bulan ke belakang (total 12 bulan)
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -11, 0)
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, -1)
		return start, end, nil
	case "year":
		// Bulan berjalan dan bulan ke belakang s/d awal tahun yaitu Januari
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, -1)
		return start, end, nil
	case "custom":
		return parseCustomRange(customStart, customEnd)
	default:
		return time.Time{}, time.Time{}, ErrInvalidPeriod
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (s *Service) Dashboard(ctx context.Context, companyID int64, periodCode, customStart, customEnd string) (*DashboardResponse, error) {
	start, end, err := resolvePeriod(periodCode, customStart, customEnd, time.Now())
	if err != nil {
		return nil, err
	}

	metric, err := s.repo.Dashboard(ctx, companyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("dashboard metric: %w", err)
	}

	return &DashboardResponse{
		CompanyID: companyID,
		Period: PeriodInfo{
			Code:  orDefault(periodCode, "all"),
			Start: start.Format(dateFmt),
			End:   end.Format(dateFmt),
		},
		CostSaving: Metric{
			Title:   "Cost Savings (Rp)",
			Value:   metric.CostSaving,
			Unit:    "Million (Juta)",
			Remarks: "Nilai ekonomis hasil negoisasi PR vs PO",
		},
		SavingRate: Metric{
			Title:   "% Savings",
			Value:   metric.SavingRate,
			Unit:    "% [Saving Rate]",
			Remarks: "Persentase saving hasil negoisasi PR vs PO",
		},
		OTDRate: Metric{
			Title:   "On-Time Delivery Rate (OTD)",
			Value:   metric.OTDRate,
			Unit:    "% [OTD Rate]",
			Remarks: "Persentase pesanan tepat waktu",
		},
		POCycleTime: Metric{
			Title:   "Average PO Cycle Time",
			Value:   metric.AvgCycleDays,
			Unit:    "Hari",
			Remarks: "Rata-rata waktu penerbitan PO dari PR diselesaikan",
		},
	}, nil
}

func (s *Service) POCycleTimeTrend(ctx context.Context, companyID int64, periodCode, customStart, customEnd string) ([]TrendPoint, error) {
	start, end, err := resolvePOCycleTimeTrendPeriod(periodCode, customStart, customEnd, time.Now())
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.POCycleTimeTrend(ctx, companyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("po cycle time trend: %w", err)
	}
	result := make([]TrendPoint, len(rows))
	for i, r := range rows {
		result[i] = TrendPoint{Label: r.Month, Value: r.AvgCycleDays}
	}
	return result, nil
}

func (s *Service) PurchaseTrendYTD(ctx context.Context, companyID int64) ([]YTDPoint, error) {
	now := time.Now()
	rows, err := s.repo.PurchaseTrendYTD(ctx, companyID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, fmt.Errorf("purchase trend ytd: %w", err)
	}
	result := make([]YTDPoint, len(rows))
	for i, r := range rows {
		result[i] = YTDPoint{Label: r.Month, YTDThisYear: r.YTDThisYear, YTDLastYear: r.YTDLastYear}
	}
	return result, nil
}
