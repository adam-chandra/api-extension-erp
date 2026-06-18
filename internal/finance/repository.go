package finance

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	// DailyAggregate returns per-day per-category net amounts for [start, end].
	//
	//   net = SUM(credit−debit) for income; SUM(debit−credit) for expense.
	DailyAggregate(ctx context.Context, companyID int64, start, end time.Time) ([]dailyRow, error)
	// TopReturLines returns up to `limit` retur penjualan lines, ordered by absolute amount desc.
	TopReturLines(ctx context.Context, companyID int64, limit int) ([]returRow, error)
	// ReturBalanceByAccount returns per-COA balance (within [start, end]) for retur accounts.
	ReturBalanceByAccount(ctx context.Context, companyID int64, start, end time.Time) ([]returAccountRow, error)
}

type dailyRow struct {
	Date     time.Time `gorm:"column:date"`
	Category string    `gorm:"column:category"`
	Net      float64   `gorm:"column:net"`
}

type returRow struct {
	Description string    `gorm:"column:description"`
	Date        time.Time `gorm:"column:date"`
	Amount      float64   `gorm:"column:amount"`
}

type returAccountRow struct {
	Code    string  `gorm:"column:code"`
	Name    string  `gorm:"column:name"`
	Balance float64 `gorm:"column:balance"`
}

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) DailyAggregate(ctx context.Context, companyID int64, start, end time.Time) ([]dailyRow, error) {
	var rows []dailyRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT b.date,
		            a.category,
		            SUM(
		                CASE a.internal_group
		                    WHEN 'income'  THEN b.credit - b.debit
		                    WHEN 'expense' THEN b.debit  - b.credit
		                    ELSE                b.debit  - b.credit
		                END
		            )::NUMERIC(20,2) AS net
		     FROM finance.account_daily_balance b
		     JOIN finance.v_account_classified  a ON a.source_id = b.account_source_id
		     WHERE b.company_source_id = ?
		       AND b.date >= ? AND b.date <= ?
		     GROUP BY b.date, a.category
		     ORDER BY b.date`,
			companyID, start, end).
		Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) TopReturLines(ctx context.Context, companyID int64, limit int) ([]returRow, error) {
	var rows []returRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT description, date, ABS(amount) AS amount
		     FROM finance.retur_lines
		     WHERE company_source_id = ?
		     ORDER BY ABS(amount) DESC, date DESC
		     LIMIT ?`,
			companyID, limit).
		Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) ReturBalanceByAccount(ctx context.Context, companyID int64, start, end time.Time) ([]returAccountRow, error) {
	var rows []returAccountRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT a.code,
		            MIN(a.name) AS name,
		            ABS(SUM(b.debit - b.credit))::NUMERIC(20,2) AS balance
		     FROM finance.account_daily_balance b
		     JOIN finance.accounts a ON a.source_id = b.account_source_id
		     WHERE b.company_source_id = ?
		       AND b.date >= ? AND b.date <= ?
		       AND (a.code LIKE '40300%' OR a.code LIKE '40400%')
		     GROUP BY a.code
		     HAVING ABS(SUM(b.debit - b.credit)) > 0
		     ORDER BY a.code`,
			companyID, start, end).
		Scan(&rows).Error
	return rows, err
}
