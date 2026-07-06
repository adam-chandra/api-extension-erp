package procurement

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Dashboard(ctx context.Context, companyID int64, start, end time.Time) (*metricRow, error)
	POCycleTimeTrend(ctx context.Context, companyID int64, start, end time.Time) ([]trendRow, error)
	PurchaseTrendYTD(ctx context.Context, companyID int64, year, untilMonth int) ([]ytdRow, error)
}

type metricRow struct {
	CostSaving   float64 `gorm:"column:cost_saving"`
	SavingRate   float64 `gorm:"column:saving_rate"`
	AvgCycleDays float64 `gorm:"column:avg_cycle_days"`
	OTDRate      float64 `gorm:"column:otd_rate"`
	OTDOnTime    int64   `gorm:"column:on_time"`
	OTDTotal     int64   `gorm:"column:total"`
}

type trendRow struct {
	Month        string  `gorm:"column:month"`
	AvgCycleDays float64 `gorm:"column:avg_cycle_days"`
}

type ytdRow struct {
	Month       string  `gorm:"column:month"`
	YTDThisYear float64 `gorm:"column:ytd_this_year"`
	YTDLastYear float64 `gorm:"column:ytd_last_year"`
}

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) Dashboard(ctx context.Context, companyID int64, start, end time.Time) (*metricRow, error) {
	var kpi metricRow
	err := r.db.WithContext(ctx).Raw(`
        WITH matched_pos AS (
            SELECT
                source_id,
                company_source_id,
                amount_untaxed,
                origin
            FROM procurement.purchase_orders
            WHERE company_source_id = ?
              AND date_order BETWEEN ? AND ?
              AND from_purchase_request = true
              AND state IN ('purchase', 'done')
        ),
        matched_po_pr AS (
            SELECT DISTINCT
                po.source_id AS po_source_id,
                po.amount_untaxed,
                pr.source_id AS pr_source_id,
                pr.amount_total
            FROM matched_pos po
            JOIN procurement.purchase_order_purchase_requests rel ON rel.purchase_order_source_id = po.source_id
            JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
                AND pr.company_source_id = po.company_source_id
                AND pr.state NOT IN ('cancel', 'rejected')
            UNION
            SELECT DISTINCT
                po.source_id AS po_source_id,
                po.amount_untaxed,
                pr.source_id AS pr_source_id,
                pr.amount_total
            FROM matched_pos po
            JOIN procurement.purchase_requests pr ON pr.name = po.origin
                AND pr.company_source_id = po.company_source_id
                AND pr.state NOT IN ('cancel', 'rejected')
        ),
        po_summary AS (
            SELECT COALESCE(SUM(amount_untaxed), 0) AS po_val
            FROM (
                SELECT DISTINCT po_source_id, amount_untaxed
                FROM matched_po_pr
            ) sub
        ),
        cycle_summary AS (
            SELECT
                COALESCE(ROUND(AVG(EXTRACT(EPOCH FROM (date_approve::timestamp - pr_confirm_date)) / 86400), 1), 0) AS avg_cycle_days
            FROM procurement.purchase_orders
            WHERE company_source_id = ?
              AND from_purchase_request = true
              AND state IN ('purchase', 'done')
              AND date_approve IS NOT NULL
              AND pr_confirm_date IS NOT NULL
              AND date_approve BETWEEN ? AND ?
        ),
        matched_prs AS (
            SELECT DISTINCT
                pr_source_id,
                amount_total
            FROM matched_po_pr
        ),
        pr_summary AS (
            SELECT COALESCE(SUM(amount_total), 0) AS pr_val
            FROM matched_prs
        )
        SELECT
            ROUND((pr_summary.pr_val - po_summary.po_val) / 1000000, 2) AS cost_saving,
            CASE
                WHEN pr_summary.pr_val > 0
                THEN ROUND(((pr_summary.pr_val - po_summary.po_val) / pr_summary.pr_val) * 100, 1)
                ELSE 0
            END AS saving_rate,
            cycle_summary.avg_cycle_days AS avg_cycle_days
        FROM pr_summary, po_summary, cycle_summary
    `, companyID, start, end, companyID, start, end).Scan(&kpi).Error
	if err != nil {
		return nil, err
	}

	var otd struct {
		OnTime int64   `gorm:"column:on_time"`
		Total  int64   `gorm:"column:total"`
		Rate   float64 `gorm:"column:otd_rate"`
	}
	err = r.db.WithContext(ctx).Raw(`
        SELECT
            COUNT(*) FILTER (WHERE gr.date_done <= po.date_planned) AS on_time,
            COUNT(*) AS total,
            COALESCE(ROUND(COUNT(*) FILTER (WHERE gr.date_done <= po.date_planned)::NUMERIC / NULLIF(COUNT(*), 0) * 100, 1), 0) AS otd_rate
        FROM procurement.goods_receipts gr
        JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id
        WHERE gr.company_source_id = ?
          AND gr.state = 'done'
          AND gr.date_done >= ?
          AND gr.date_done < ?
          AND NOT (COALESCE(po.is_goods_orders, false) = true AND COALESCE(po.product_type, '') = 'storable')
    `, companyID, start, end.AddDate(0, 0, 1)).Scan(&otd).Error
	if err != nil {
		return nil, err
	}

	kpi.OTDRate = otd.Rate
	kpi.OTDOnTime = otd.OnTime
	kpi.OTDTotal = otd.Total
	return &kpi, nil
}

func (r *gormRepo) POCycleTimeTrend(ctx context.Context, companyID int64, start, end time.Time) ([]trendRow, error) {
	var rows []trendRow
	err := r.db.WithContext(ctx).Raw(`
        WITH months AS (
            SELECT generate_series(
                DATE_TRUNC('month', ?::timestamp),
                DATE_TRUNC('month', ?::timestamp),
                INTERVAL '1 month'
            )::date AS month
        ),
        monthly AS (
            SELECT
                DATE_TRUNC('month', date_approve)::date AS month,
                ROUND(AVG(EXTRACT(EPOCH FROM (date_approve::timestamp - pr_confirm_date)) / 86400), 1) AS avg_cycle_days
            FROM procurement.purchase_orders
            WHERE company_source_id = ?
              AND state IN ('purchase', 'done')
              AND date_approve IS NOT NULL
              AND pr_confirm_date IS NOT NULL
              AND date_approve BETWEEN ? AND ?
              AND from_purchase_request = true
            GROUP BY DATE_TRUNC('month', date_approve)::date
        )
        SELECT
            TO_CHAR(months.month, 'Mon') AS month,
            COALESCE(monthly.avg_cycle_days, 0) AS avg_cycle_days
        FROM months
        LEFT JOIN monthly ON monthly.month = months.month
        ORDER BY months.month
    `, start, end, companyID, start, end).Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) PurchaseTrendYTD(ctx context.Context, companyID int64, year, untilMonth int) ([]ytdRow, error) {
	var rows []ytdRow
	err := r.db.WithContext(ctx).Raw(`
        WITH months AS (
            SELECT generate_series(1, ?) AS mon
        ),
        monthly AS (
            SELECT
				EXTRACT(MONTH FROM date_order)::int    AS mon,
				EXTRACT(YEAR FROM date_order)::int    AS yr,
				SUM(amount_untaxed)                   AS amount
			FROM procurement.purchase_orders
			WHERE company_source_id = ?
			  AND state IN ('purchase', 'done')
			  AND EXTRACT(YEAR FROM date_order) IN (?, ?)
			GROUP BY 1, 2
		),
		current_year AS (
			SELECT
				months.mon,
				ROUND(SUM(COALESCE(monthly.amount, 0)) OVER (ORDER BY months.mon) / 1000000, 2) AS ytd
			FROM months
			LEFT JOIN monthly ON monthly.mon = months.mon AND monthly.yr = ?
		),
		prev_year AS (
			SELECT
				months.mon,
				ROUND(SUM(COALESCE(monthly.amount, 0)) OVER (ORDER BY months.mon) / 1000000, 2) AS ytd
			FROM months
			LEFT JOIN monthly ON monthly.mon = months.mon AND monthly.yr = ?
		)
		SELECT
			TO_CHAR(MAKE_DATE(?, cy.mon, 1), 'Mon') AS month,
			COALESCE(cy.ytd, 0)         AS ytd_this_year,
			COALESCE(py.ytd, 0)         AS ytd_last_year
		FROM current_year cy
		LEFT JOIN prev_year py ON py.mon = cy.mon
		ORDER BY cy.mon
	`, untilMonth, companyID, year, year-1, year, year-1, year).Scan(&rows).Error
	return rows, err
}
