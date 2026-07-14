package finance

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	DailyAggregate(ctx context.Context, companyID int64, start, end time.Time) ([]dailyRow, error)
	TopReturLines(ctx context.Context, companyID int64, limit int) ([]returRow, error)
	ReturBalanceByAccount(ctx context.Context, companyID int64, start, end time.Time) ([]returAccountRow, error)

	GetConsolidatedBalance(ctx context.Context, companyIDs []int64, start, end time.Time, coaPrefixes []string) ([]consolidatedBalanceRow, error)
	GetEliminationEntries(ctx context.Context, companyIDs []int64, start, end time.Time) ([]eliminationRow, error)

	GetFinancialReportStructure(ctx context.Context, reportType string) ([]reportLineRow, error)
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

type consolidatedBalanceRow struct {
	CompanySourceID   int64   `gorm:"column:company_source_id"`
	CompanyName       string  `gorm:"column:company_name"`
	AccountSourceID   int64   `gorm:"column:account_source_id"`
	AccountCode       string  `gorm:"column:account_code"`
	AccountName       string  `gorm:"column:account_name"`
	AccountType       string  `gorm:"column:account_type"`
	Debit             float64 `gorm:"column:debit"`
	Credit            float64 `gorm:"column:credit"`
	Balance           float64 `gorm:"column:balance"`
	EliminationDebit  float64 `gorm:"column:elimination_debit"`
	EliminationCredit float64 `gorm:"column:elimination_credit"`
}

type eliminationRow struct {
	ID                      int64   `gorm:"column:id"`
	Date                    string  `gorm:"column:date"`
	CompanySourceID         int64   `gorm:"column:company_source_id"`
	CompanyName             string  `gorm:"column:company_name"`
	AccountCode             string  `gorm:"column:account_code"`
	AccountName             string  `gorm:"column:account_name"`
	PartnerName             string  `gorm:"column:partner_name"`
	IntercompanyPartnerName string  `gorm:"column:intercompany_partner_name"`
	Description             string  `gorm:"column:description"`
	Debit                   float64 `gorm:"column:debit"`
	Credit                  float64 `gorm:"column:credit"`
}

type glEntryRow struct {
	ID               int64   `gorm:"column:id"`
	Date             string  `gorm:"column:date"`
	CompanyName      string  `gorm:"column:company_name"`
	JournalName      string  `gorm:"column:journal_name"`
	PartnerName      string  `gorm:"column:partner_name"`
	MoveName         string  `gorm:"column:move_name"`
	EntryLabel       string  `gorm:"column:entry_label"`
	Debit            float64 `gorm:"column:debit"`
	Credit           float64 `gorm:"column:credit"`
	Balance          float64 `gorm:"column:balance"`
	IsIntercompany   bool    `gorm:"column:is_intercompany"`
	IntercompanyWith string  `gorm:"column:intercompany_with"`
}

type reportLineRow struct {
	SourceID       int64  `gorm:"column:source_id"`
	ParentSourceID *int64 `gorm:"column:parent_source_id"`
	ReportSourceID *int64 `gorm:"column:report_source_id"`
	Name           string `gorm:"column:name"`
	Code           string `gorm:"column:code"`
	Sequence       int    `gorm:"column:sequence"`
	Level          int    `gorm:"column:level"`
	Domain         string `gorm:"column:domain"`
	LineType       string `gorm:"column:line_type"`
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

func (r *gormRepo) GetConsolidatedBalance(ctx context.Context, companyIDs []int64, start, end time.Time, coaPrefixes []string) ([]consolidatedBalanceRow, error) {
	var rows []consolidatedBalanceRow

	coaConditions := make([]string, 0, len(coaPrefixes))
	for _, p := range coaPrefixes {
		coaConditions = append(coaConditions, "a.code LIKE '"+p+"%'")
	}
	coaFilter := strings.Join(coaConditions, " OR ")
	if coaFilter == "" {
		coaFilter = "TRUE"
	}

	isBalanceSheet := len(coaPrefixes) > 0 && coaPrefixes[0] == "1"

	var query string
	var args []interface{}

	if isBalanceSheet {
		// Balance Sheet: kumulatif s/d end date, filter balance != 0
		query = `SELECT
			ml.company_source_id,
			c.name AS company_name,
			ml.account_source_id,
			a.code AS account_code,
			a.name AS account_name,
			COALESCE(a.account_type, '') AS account_type,
			SUM(ml.debit)  AS debit,
			SUM(ml.credit) AS credit,
			CASE
				WHEN COALESCE(a.account_type,'') IN (
					'asset_receivable','asset_cash','asset_current','asset_non_current',
					'asset_prepayments','asset_fixed'
				) THEN SUM(ml.debit - ml.credit)
				WHEN COALESCE(a.account_type,'') IN (
					'expense','expense_depreciation','expense_amortization',
					'expense_direct_cost','expense_other'
				) THEN SUM(ml.credit - ml.debit)
				WHEN COALESCE(a.account_type,'') = 'expense_tax' THEN SUM(ml.debit - ml.credit)
				WHEN COALESCE(a.account_type,'') <> '' THEN SUM(ml.credit - ml.debit)
				WHEN LEFT(a.code,1) IN ('1') THEN SUM(ml.debit - ml.credit)
				WHEN LEFT(a.code,1) = '6'    THEN SUM(ml.credit - ml.debit)
				WHEN LEFT(a.code,1) IN ('5','8') THEN SUM(ml.debit - ml.credit)
				ELSE SUM(ml.credit - ml.debit)
			END AS balance,
			SUM(CASE WHEN ml.is_intercompany THEN ml.debit ELSE 0 END)  AS elimination_debit,
			SUM(CASE WHEN ml.is_intercompany THEN ml.credit ELSE 0 END) AS elimination_credit
		FROM finance.move_lines ml
		JOIN finance.accounts a ON a.source_id = ml.account_source_id
		JOIN auth.companies c ON c.source_id = ml.company_source_id
		WHERE ml.company_source_id IN ?
		  AND ml.date <= ?
		  AND ml.move_state = 'posted'
		  AND (` + coaFilter + `)
		GROUP BY ml.company_source_id, c.name, ml.account_source_id, a.code, a.name, a.account_type
		HAVING SUM(ml.debit) <> 0 OR SUM(ml.credit) <> 0
		ORDER BY a.code, c.name`
		args = []interface{}{companyIDs, end}
	} else {
		// P&L: movement dalam period saja
		query = `SELECT
			ml.company_source_id,
			c.name AS company_name,
			ml.account_source_id,
			a.code AS account_code,
			a.name AS account_name,
			COALESCE(a.account_type, '') AS account_type,
			SUM(ml.debit)  AS debit,
			SUM(ml.credit) AS credit,
			CASE
				WHEN COALESCE(a.account_type,'') IN (
					'asset_receivable','asset_cash','asset_current','asset_non_current',
					'asset_prepayments','asset_fixed'
				) THEN SUM(ml.debit - ml.credit)
				WHEN COALESCE(a.account_type,'') IN (
					'income', 'income_other', 'income_interest'
				) THEN SUM(ml.credit - ml.debit)
				WHEN COALESCE(a.account_type,'') IN (
					'expense','expense_depreciation','expense_amortization',
					'expense_direct_cost','expense_other'
				) THEN SUM(ml.credit - ml.debit)
				WHEN COALESCE(a.account_type,'') = 'expense_tax' THEN SUM(ml.debit - ml.credit)
				WHEN COALESCE(a.account_type,'') <> '' THEN SUM(ml.credit - ml.debit)
				WHEN LEFT(a.code,1) IN ('1') THEN SUM(ml.debit - ml.credit)
				WHEN LEFT(a.code,1) = '6'    THEN SUM(ml.credit - ml.debit)
				WHEN LEFT(a.code,1) IN ('5','8') THEN SUM(ml.debit - ml.credit)
				ELSE SUM(ml.credit - ml.debit)
			END AS balance,
			SUM(CASE WHEN ml.is_intercompany THEN ml.debit ELSE 0 END)  AS elimination_debit,
			SUM(CASE WHEN ml.is_intercompany THEN ml.credit ELSE 0 END) AS elimination_credit
		FROM finance.move_lines ml
		JOIN finance.accounts a ON a.source_id = ml.account_source_id
		JOIN auth.companies c ON c.source_id = ml.company_source_id
		WHERE ml.company_source_id IN ?
		  AND ml.date >= ? AND ml.date <= ?
		  AND ml.move_state = 'posted'
		  AND (` + coaFilter + `)
		GROUP BY ml.company_source_id, c.name, ml.account_source_id, a.code, a.name, a.account_type
		HAVING SUM(ml.debit) <> 0 OR SUM(ml.credit) <> 0
		ORDER BY a.code, c.name`
		args = []interface{}{companyIDs, start, end}
	}

	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) GetEliminationEntries(ctx context.Context, companyIDs []int64, start, end time.Time) ([]eliminationRow, error) {
	var rows []eliminationRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT
		        ml.id,
		        ml.date::TEXT AS date,
		        ml.company_source_id,
		        c.name AS company_name,
		        a.code AS account_code,
		        a.name AS account_name,
		        COALESCE(p.name, '') AS partner_name,
		        COALESCE(ic.name, '') AS intercompany_partner_name,
		        ml.name AS description,
		        ml.debit,
		        ml.credit
		    FROM finance.move_lines ml
		    JOIN finance.accounts a ON a.source_id = ml.account_source_id
		    JOIN auth.companies c ON c.source_id = ml.company_source_id
		    LEFT JOIN auth.partners p ON p.source_id = ml.partner_source_id
		    LEFT JOIN auth.companies ic ON ic.source_id = ml.intercompany_partner_id
		    WHERE ml.company_source_id IN ?
		      AND ml.date >= ? AND ml.date <= ?
		      AND ml.is_intercompany = TRUE
		      AND ml.move_state = 'posted'
		    ORDER BY ml.date DESC, ml.id`,
			companyIDs, start, end).
		Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) GetFinancialReportStructure(ctx context.Context, reportType string) ([]reportLineRow, error) {
	var rows []reportLineRow
	// financial_reports tabel kosong — ambil semua lines langsung.
	// Service layer yang pilih root berdasarkan code (BS/PL).
	err := r.db.WithContext(ctx).
		Raw(`SELECT
		        source_id,
		        parent_source_id,
		        report_source_id,
		        name,
		        COALESCE(code, '')      AS code,
		        COALESCE(sequence, 0)   AS sequence,
		        COALESCE(level, 0)      AS level,
		        COALESCE(domain, '')    AS domain,
		        COALESCE(line_type, '') AS line_type
		    FROM finance.financial_report_lines
		    ORDER BY sequence, level, source_id`).
		Scan(&rows).Error
	return rows, err
}
