// Package finance — white-box tests (same package) so we can reach
// unexported helpers (resolvePeriod, monthRange, orDefault) and internal types.
package finance

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test double
// ---------------------------------------------------------------------------

type mockRepo struct {
	dailyRows        []dailyRow
	dailyErr         error
	returRows        []returRow
	returErr         error
	returAccountRows []returAccountRow
	returAccountErr  error
}

func (m *mockRepo) DailyAggregate(_ context.Context, _ int64, _, _ time.Time) ([]dailyRow, error) {
	return m.dailyRows, m.dailyErr
}

func (m *mockRepo) TopReturLines(_ context.Context, _ int64, _ int) ([]returRow, error) {
	return m.returRows, m.returErr
}

func (m *mockRepo) ReturBalanceByAccount(_ context.Context, _ int64, _, _ time.Time) ([]returAccountRow, error) {
	return m.returAccountRows, m.returAccountErr
}

// ---------------------------------------------------------------------------
// resolvePeriod
// ---------------------------------------------------------------------------

func TestResolvePeriod_All(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	s, e, err := resolvePeriod("all", "", "", now)
	if err != nil {
		t.Fatalf("resolvePeriod(all) error = %v", err)
	}
	if s.IsZero() || e.IsZero() {
		t.Error("resolvePeriod(all) returned zero times")
	}
	if s.After(e) {
		t.Errorf("resolvePeriod(all) start %v is after end %v", s, e)
	}
}

func TestResolvePeriod_Empty_TreatedAsAll(t *testing.T) {
	now := time.Now()
	s, e, err := resolvePeriod("", "", "", now)
	if err != nil {
		t.Fatalf("resolvePeriod('') error = %v", err)
	}
	if s.IsZero() || e.IsZero() {
		t.Error("resolvePeriod('') returned zero times")
	}
}

func TestResolvePeriod_Year(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	s, e, err := resolvePeriod("year", "", "", now)
	if err != nil {
		t.Fatalf("resolvePeriod(year) error = %v", err)
	}
	if s.Month() != time.January || s.Day() != 1 {
		t.Errorf("resolvePeriod(year) start = %v, want 2025-01-01", s)
	}
	if e.Month() != time.December || e.Day() != 31 {
		t.Errorf("resolvePeriod(year) end = %v, want 2025-12-31", e)
	}
}

func TestResolvePeriod_Month(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	s, e, err := resolvePeriod("month", "", "", now)
	if err != nil {
		t.Fatalf("resolvePeriod(month) error = %v", err)
	}
	if s.Day() != 1 || s.Month() != time.March {
		t.Errorf("resolvePeriod(month) start = %v, want 2025-03-01", s)
	}
	if e.Month() != time.March {
		t.Errorf("resolvePeriod(month) end = %v, want end of March", e)
	}
}

func TestResolvePeriod_Custom_Valid(t *testing.T) {
	s, e, err := resolvePeriod("custom", "2025-01-01", "2025-06-30", time.Now())
	if err != nil {
		t.Fatalf("resolvePeriod(custom) error = %v", err)
	}
	if s.Format(dateFmt) != "2025-01-01" {
		t.Errorf("resolvePeriod(custom) start = %v, want 2025-01-01", s)
	}
	if e.Format(dateFmt) != "2025-06-30" {
		t.Errorf("resolvePeriod(custom) end = %v, want 2025-06-30", e)
	}
}

func TestResolvePeriod_Custom_MissingDates(t *testing.T) {
	_, _, err := resolvePeriod("custom", "", "", time.Now())
	if !errors.Is(err, ErrInvalidRange) {
		t.Errorf("resolvePeriod(custom) missing dates = %v, want ErrInvalidRange", err)
	}
}

func TestResolvePeriod_Custom_MissingEnd(t *testing.T) {
	_, _, err := resolvePeriod("custom", "2025-01-01", "", time.Now())
	if !errors.Is(err, ErrInvalidRange) {
		t.Errorf("resolvePeriod(custom) missing end = %v, want ErrInvalidRange", err)
	}
}

func TestResolvePeriod_Custom_InvalidStart(t *testing.T) {
	_, _, err := resolvePeriod("custom", "not-a-date", "2025-06-30", time.Now())
	if !errors.Is(err, ErrInvalidRange) {
		t.Errorf("resolvePeriod(custom) invalid start = %v, want ErrInvalidRange", err)
	}
}

func TestResolvePeriod_Custom_InvalidEnd(t *testing.T) {
	_, _, err := resolvePeriod("custom", "2025-01-01", "bad-date", time.Now())
	if !errors.Is(err, ErrInvalidRange) {
		t.Errorf("resolvePeriod(custom) invalid end = %v, want ErrInvalidRange", err)
	}
}

func TestResolvePeriod_Custom_EndBeforeStart(t *testing.T) {
	_, _, err := resolvePeriod("custom", "2025-06-01", "2025-01-01", time.Now())
	if !errors.Is(err, ErrInvalidRange) {
		t.Errorf("resolvePeriod(custom) end < start = %v, want ErrInvalidRange", err)
	}
}

func TestResolvePeriod_Unknown(t *testing.T) {
	_, _, err := resolvePeriod("quarterly", "", "", time.Now())
	if !errors.Is(err, ErrInvalidPeriod) {
		t.Errorf("resolvePeriod(unknown) = %v, want ErrInvalidPeriod", err)
	}
}

// ---------------------------------------------------------------------------
// orDefault
// ---------------------------------------------------------------------------

func TestOrDefault_NonEmpty(t *testing.T) {
	if got := orDefault("hello", "world"); got != "hello" {
		t.Errorf("orDefault(hello) = %q, want hello", got)
	}
}

func TestOrDefault_Empty(t *testing.T) {
	if got := orDefault("", "world"); got != "world" {
		t.Errorf("orDefault('') = %q, want world", got)
	}
}

// ---------------------------------------------------------------------------
// monthRange
// ---------------------------------------------------------------------------

func TestMonthRange_SingleMonth(t *testing.T) {
	start := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	result := monthRange(start, end, nil)
	if len(result) != 1 {
		t.Errorf("monthRange single month len = %d, want 1", len(result))
	}
	if result[0].Month != "2025-03" {
		t.Errorf("monthRange single month = %q, want 2025-03", result[0].Month)
	}
}

func TestMonthRange_ThreeMonths(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	existing := map[string]*TrendPoint{
		"2025-02": {Month: "2025-02", SalesNet: 5000},
	}
	result := monthRange(start, end, existing)
	if len(result) != 3 {
		t.Errorf("monthRange 3 months len = %d, want 3", len(result))
	}
	// 2025-01 and 2025-03 should be zero-value TrendPoints (gap-filling)
	if result[0].SalesNet != 0 {
		t.Error("monthRange should fill gaps with zero SalesNet")
	}
	if result[1].SalesNet != 5000 {
		t.Errorf("monthRange existing month SalesNet = %f, want 5000", result[1].SalesNet)
	}
}

// ---------------------------------------------------------------------------
// Service.Returns
// ---------------------------------------------------------------------------

func TestReturns_Success(t *testing.T) {
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		returRows: []returRow{
			{Description: "Item A", Date: date, Amount: 6000000},
			{Description: "Item B", Date: date, Amount: 4000000},
		},
	}
	svc := NewService(repo)
	lines, err := svc.Returns(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("Returns() error = %v", err)
	}
	if len(lines) != 2 {
		t.Errorf("Returns() len = %d, want 2", len(lines))
	}
	if lines[0].No != 1 || lines[1].No != 2 {
		t.Error("Returns() should number rows starting from 1")
	}
	if lines[0].Percentage == 0 {
		t.Error("Returns() percentage should be non-zero when total > 0")
	}
	// First item has more amount → higher percentage
	if lines[0].Percentage <= lines[1].Percentage {
		t.Error("Returns() first item should have higher percentage")
	}
}

func TestReturns_ZeroTotalAmount(t *testing.T) {
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		returRows: []returRow{{Description: "Zero Item", Date: date, Amount: 0}},
	}
	svc := NewService(repo)
	lines, err := svc.Returns(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("Returns() error = %v", err)
	}
	if lines[0].Percentage != 0 {
		t.Error("Returns() percentage should be 0 when total is 0")
	}
}

func TestReturns_DefaultLimit_WhenZero(t *testing.T) {
	repo := &mockRepo{returRows: []returRow{}}
	svc := NewService(repo)
	_, err := svc.Returns(context.Background(), 1, 0)
	if err != nil {
		t.Fatalf("Returns() limit=0 error = %v", err)
	}
}

func TestReturns_DefaultLimit_WhenOverMax(t *testing.T) {
	repo := &mockRepo{returRows: []returRow{}}
	svc := NewService(repo)
	_, err := svc.Returns(context.Background(), 1, 9999)
	if err != nil {
		t.Fatalf("Returns() limit=9999 error = %v", err)
	}
}

func TestReturns_RepoError(t *testing.T) {
	repo := &mockRepo{returErr: errors.New("db connection failed")}
	svc := NewService(repo)
	_, err := svc.Returns(context.Background(), 1, 10)
	if err == nil {
		t.Error("Returns() should propagate repo error")
	}
}

// ---------------------------------------------------------------------------
// Service.ReturnsByAccount
// ---------------------------------------------------------------------------

func TestReturnsByAccount_Success(t *testing.T) {
	repo := &mockRepo{
		returAccountRows: []returAccountRow{
			{Code: "40300", Name: "Retur A", Balance: 2000000},
			{Code: "40400", Name: "Retur B", Balance: 1000000},
			{Code: "40500", Name: "Retur C", Balance: 500000},
		},
	}
	svc := NewService(repo)
	rows, err := svc.ReturnsByAccount(context.Background(), 1, "year", "", "", 10)
	if err != nil {
		t.Fatalf("ReturnsByAccount() error = %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("ReturnsByAccount() len = %d, want 3", len(rows))
	}
	// Should be sorted descending by balance
	if rows[0].Balance < rows[1].Balance {
		t.Error("ReturnsByAccount() should sort by balance descending")
	}
}

func TestReturnsByAccount_LimitTruncation(t *testing.T) {
	rows := make([]returAccountRow, 5)
	for i := range rows {
		rows[i] = returAccountRow{Code: "400", Name: "Retur", Balance: float64(i+1) * 1000}
	}
	repo := &mockRepo{returAccountRows: rows}
	svc := NewService(repo)
	result, err := svc.ReturnsByAccount(context.Background(), 1, "year", "", "", 2)
	if err != nil {
		t.Fatalf("ReturnsByAccount() limit error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("ReturnsByAccount() truncated len = %d, want 2", len(result))
	}
}

func TestReturnsByAccount_DefaultLimit_WhenInvalid(t *testing.T) {
	repo := &mockRepo{returAccountRows: []returAccountRow{}}
	svc := NewService(repo)
	_, err := svc.ReturnsByAccount(context.Background(), 1, "year", "", "", 0)
	if err != nil {
		t.Fatalf("ReturnsByAccount() limit=0 error = %v", err)
	}
}

func TestReturnsByAccount_InvalidPeriod(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	_, err := svc.ReturnsByAccount(context.Background(), 1, "bad-period", "", "", 10)
	if err == nil {
		t.Error("ReturnsByAccount() should return error for invalid period")
	}
}

func TestReturnsByAccount_RepoError(t *testing.T) {
	repo := &mockRepo{returAccountErr: errors.New("db error")}
	svc := NewService(repo)
	_, err := svc.ReturnsByAccount(context.Background(), 1, "year", "", "", 10)
	if err == nil {
		t.Error("ReturnsByAccount() should propagate repo error")
	}
}

func TestReturnsByAccount_ZeroTotal_Percentage(t *testing.T) {
	repo := &mockRepo{
		returAccountRows: []returAccountRow{
			{Code: "400", Name: "Retur", Balance: 0},
		},
	}
	svc := NewService(repo)
	rows, err := svc.ReturnsByAccount(context.Background(), 1, "year", "", "", 10)
	if err != nil {
		t.Fatalf("ReturnsByAccount() zero total error = %v", err)
	}
	if len(rows) > 0 && rows[0].Percentage != 0 {
		t.Error("ReturnsByAccount() percentage should be 0 when total is 0")
	}
}

// ---------------------------------------------------------------------------
// Service.Dashboard
// ---------------------------------------------------------------------------

func TestDashboard_Success(t *testing.T) {
	date := time.Date(2025, 4, 10, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		dailyRows: []dailyRow{
			{Date: date, Category: "revenue_operating", Net: 10000000},
			{Date: date, Category: "revenue_non_operating", Net: 1000000},
			{Date: date, Category: "cost_of_revenue", Net: 4000000},
			{Date: date, Category: "sales_discount", Net: -300000},
			{Date: date, Category: "sales_return", Net: -200000},
		},
	}
	svc := NewService(repo)
	resp, err := svc.Dashboard(context.Background(), 1, "year", "", "")
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}
	if resp.KPI.CostOfRevenue != 4000000 {
		t.Errorf("Dashboard() CostOfRevenue = %f, want 4000000", resp.KPI.CostOfRevenue)
	}
	if resp.KPI.SalesDiscount != 300000 {
		t.Errorf("Dashboard() SalesDiscount = %f, want 300000", resp.KPI.SalesDiscount)
	}
	if resp.KPI.SalesReturn != 200000 {
		t.Errorf("Dashboard() SalesReturn = %f, want 200000", resp.KPI.SalesReturn)
	}
	if len(resp.Trend) == 0 {
		t.Error("Dashboard() Trend should not be empty")
	}
}

func TestDashboard_EmptyRows(t *testing.T) {
	repo := &mockRepo{dailyRows: []dailyRow{}}
	svc := NewService(repo)
	resp, err := svc.Dashboard(context.Background(), 1, "month", "", "")
	if err != nil {
		t.Fatalf("Dashboard() empty rows error = %v", err)
	}
	if resp.KPI.SalesNet != 0 || resp.KPI.CostOfRevenue != 0 {
		t.Error("Dashboard() empty rows: KPI should be zero")
	}
}

func TestDashboard_InvalidPeriod(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	_, err := svc.Dashboard(context.Background(), 1, "unknown-period", "", "")
	if err == nil {
		t.Error("Dashboard() should return error for invalid period")
	}
}

func TestDashboard_RepoError(t *testing.T) {
	repo := &mockRepo{dailyErr: errors.New("db error")}
	svc := NewService(repo)
	_, err := svc.Dashboard(context.Background(), 1, "year", "", "")
	if err == nil {
		t.Error("Dashboard() should propagate repo error")
	}
}

func TestDashboard_PeriodCodeInResponse(t *testing.T) {
	repo := &mockRepo{dailyRows: []dailyRow{}}
	svc := NewService(repo)

	resp, _ := svc.Dashboard(context.Background(), 1, "", "", "")
	if resp.Period.Code != "all" {
		t.Errorf("Dashboard() empty period code = %q, want all", resp.Period.Code)
	}
}

func TestDashboard_CustomPeriod(t *testing.T) {
	repo := &mockRepo{dailyRows: []dailyRow{}}
	svc := NewService(repo)
	resp, err := svc.Dashboard(context.Background(), 1, "custom", "2025-01-01", "2025-03-31")
	if err != nil {
		t.Fatalf("Dashboard() custom period error = %v", err)
	}
	if resp.Period.Start != "2025-01-01" {
		t.Errorf("Dashboard() Period.Start = %q, want 2025-01-01", resp.Period.Start)
	}
}
