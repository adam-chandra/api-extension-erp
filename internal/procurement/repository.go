package procurement

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	poDateOrderDesc = "po.date_order DESC"
	poNameILike     = "po.name ILIKE ?"
)

type Repository interface {
	Dashboard(ctx context.Context, companyID int64, start, end time.Time) (*metricRow, error)
	POCycleTimeTrend(ctx context.Context, companyID int64, start, end time.Time) ([]trendRow, error)
	ReceivingCycleTimeTrend(ctx context.Context, companyID int64, start, end time.Time) ([]trendRow, error)
	ProcurementCycleTimeTrend(ctx context.Context, companyID int64, start, end time.Time) ([]trendRow, error)
	PurchaseTrendYTD(ctx context.Context, companyID int64, year, untilMonth int) ([]ytdRow, error)
	ListDocuments(ctx context.Context, companyID int64, start, end time.Time, metricType string, search string, offset, limit int) ([]DocumentRow, int64, error)
}

type metricRow struct {
	CostSaving              float64 `gorm:"column:cost_saving"`
	SavingRate              float64 `gorm:"column:saving_rate"`
	AvgCycleDays            float64 `gorm:"column:avg_cycle_days"`
	OTDRate                 float64 `gorm:"column:otd_rate"`
	OTDOnTime               int64   `gorm:"column:on_time"`
	OTDTotal                int64   `gorm:"column:total"`
	PRNoPO                  float64 `gorm:"column:pr_no_po"`
	POIncomplete            float64 `gorm:"column:po_incomplete"`
	PONoReceiving           float64 `gorm:"column:po_no_receiving"`
	AvgReceivingCycleDays   float64 `gorm:"column:avg_receiving_cycle_days"`
	AvgProcurementCycleDays float64 `gorm:"column:avg_procurement_cycle_days"`
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
              AND LOWER(COALESCE(state, '')) IN ('purchase', 'done')
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
                AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
            UNION
            SELECT DISTINCT
                po.source_id AS po_source_id,
                po.amount_untaxed,
                pr.source_id AS pr_source_id,
                pr.amount_total
            FROM matched_pos po
            JOIN procurement.purchase_requests pr ON pr.name = po.origin
                AND pr.company_source_id = po.company_source_id
                AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
        ),
        po_summary AS (
            SELECT COALESCE(SUM(amount_untaxed), 0) AS po_val
            FROM (
                SELECT DISTINCT po_source_id, amount_untaxed FROM matched_po_pr
            ) sub
        ),
        cycle_summary AS (
            SELECT
                COALESCE(ROUND(AVG(cycle_days), 1), 0) AS avg_cycle_days
            FROM procurement.purchase_orders
            WHERE company_source_id = ?
              AND from_purchase_request = true
              AND LOWER(COALESCE(state, '')) IN ('purchase', 'done')
              AND date_approve BETWEEN ? AND ?
        ),
        pr_summary AS (
            SELECT COALESCE(SUM(amount_total), 0) AS pr_val
            FROM (
                SELECT DISTINCT pr_source_id, amount_total FROM matched_po_pr
            ) sub
        ),
        pr_pending_summary AS (
            SELECT COUNT(*) AS pr_no_po
            FROM procurement.purchase_requests pr
            WHERE pr.company_source_id = ?
              AND pr.date_start BETWEEN ? AND ?
              AND LOWER(COALESCE(pr.state, '')) IN ('purchase_request', 'purchase request')
              AND NOT EXISTS (
                  SELECT 1
                  FROM procurement.purchase_order_purchase_requests rel
                  JOIN procurement.purchase_orders po ON po.source_id = rel.purchase_order_source_id
                  WHERE rel.purchase_request_source_id = pr.source_id
                    AND po.company_source_id = pr.company_source_id
                    AND LOWER(COALESCE(po.state, '')) <> 'cancel'
              )
              AND NOT EXISTS (
                  SELECT 1
                  FROM procurement.purchase_orders po
                  WHERE po.company_source_id = pr.company_source_id
                    AND po.origin = pr.name
                    AND LOWER(COALESCE(po.state, '')) <> 'cancel'
              )
        ),
        po_incomplete_summary AS (
            SELECT COUNT(*) AS po_incomplete
            FROM procurement.purchase_orders po
            WHERE po.company_source_id = ?
              AND po.date_order BETWEEN ? AND ?
              AND LOWER(COALESCE(po.state, '')) IN (
                  'request_for_amendment',
                  'rfq',
                  'draft',
                  'rfq_sent',
                  'sent',
                  'to_approve',
                  'waiting_for_approve',
                  'waiting_for_approval',
                  'waiting_approval',
                  'rfq_approved',
                  'over_budget_approval',
                  'on_hold',
                  'blanket_ordered'
              )
        ),
        po_no_receiving_summary AS (
            SELECT COUNT(*) AS po_no_receiving
            FROM procurement.purchase_orders po
            WHERE po.company_source_id = ?
              AND po.date_order BETWEEN ? AND ?
              AND LOWER(COALESCE(po.state, '')) IN ('purchase', 'blanket_ordered')
              AND NOT EXISTS (
                  SELECT 1
                  FROM procurement.goods_receipts gr
                  WHERE gr.company_source_id = po.company_source_id
                    AND gr.po_source_id = po.source_id
                    AND LOWER(COALESCE(gr.state, '')) = 'done'
              )
        ),
        receiving_cycle_summary AS (
            SELECT
                COALESCE(ROUND(AVG(EXTRACT(EPOCH FROM (gr.date_done - po.date_approve::timestamp)) / 86400), 1), 0) AS avg_receiving_cycle_days
            FROM procurement.goods_receipts gr
            JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id
            WHERE gr.company_source_id = ?
              AND LOWER(COALESCE(gr.state, '')) = 'done'
              AND gr.date_done IS NOT NULL
              AND po.date_approve IS NOT NULL
              AND gr.date_done >= ?
              AND gr.date_done < ?
        ),
        procurement_cycle_summary AS (
            SELECT
                COALESCE(ROUND(AVG(EXTRACT(EPOCH FROM (gr.date_done - po.pr_confirm_date::timestamp)) / 86400), 1), 0) AS avg_procurement_cycle_days
            FROM procurement.goods_receipts gr
            JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id
            WHERE gr.company_source_id = ?
              AND LOWER(COALESCE(gr.state, '')) = 'done'
              AND gr.date_done IS NOT NULL
              AND po.pr_confirm_date IS NOT NULL
              AND gr.date_done >= ?
              AND gr.date_done < ?
        )
        SELECT
            ROUND((pr_summary.pr_val - po_summary.po_val) / 1000000, 2) AS cost_saving,
            CASE
                WHEN pr_summary.pr_val > 0
                THEN ROUND(((pr_summary.pr_val - po_summary.po_val) / pr_summary.pr_val) * 100, 1)
                ELSE 0
            END AS saving_rate,
            cycle_summary.avg_cycle_days AS avg_cycle_days,
            COALESCE(pr_pending_summary.pr_no_po, 0) AS pr_no_po,
            COALESCE(po_incomplete_summary.po_incomplete, 0) AS po_incomplete,
            COALESCE(po_no_receiving_summary.po_no_receiving, 0) AS po_no_receiving,
            receiving_cycle_summary.avg_receiving_cycle_days AS avg_receiving_cycle_days,
            procurement_cycle_summary.avg_procurement_cycle_days AS avg_procurement_cycle_days
        FROM pr_summary, po_summary, cycle_summary, pr_pending_summary, po_incomplete_summary, po_no_receiving_summary, receiving_cycle_summary, procurement_cycle_summary
    `, companyID, start, end, companyID, start, end, companyID, start, end, companyID, start, end, companyID, start, end, companyID, start, end.AddDate(0, 0, 1), companyID, start, end.AddDate(0, 0, 1)).Scan(&kpi).Error
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
            COUNT(*) FILTER (WHERE gr.is_on_time = true) AS on_time,
            COUNT(*) AS total,
            COALESCE(ROUND(COUNT(*) FILTER (WHERE gr.is_on_time = true)::NUMERIC / NULLIF(COUNT(*), 0) * 100, 1), 0) AS otd_rate
        FROM procurement.goods_receipts gr
        JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id
        WHERE gr.company_source_id = ?
          AND LOWER(COALESCE(gr.state, '')) = 'done'
          AND gr.date_done >= ?
          AND gr.date_done < ?
          -- Filter ini disederhanakan
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
                ROUND(AVG(cycle_days), 1) AS avg_cycle_days
            FROM procurement.purchase_orders
            WHERE company_source_id = ?
              AND LOWER(COALESCE(state, '')) IN ('purchase', 'done')
              AND date_approve IS NOT NULL
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

func (r *gormRepo) ReceivingCycleTimeTrend(ctx context.Context, companyID int64, start, end time.Time) ([]trendRow, error) {
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
                DATE_TRUNC('month', gr.date_done)::date AS month,
                ROUND(AVG(EXTRACT(EPOCH FROM (gr.date_done - po.date_approve::timestamp)) / 86400), 1) AS avg_cycle_days
            FROM procurement.goods_receipts gr
            JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id
            WHERE gr.company_source_id = ?
              AND LOWER(COALESCE(gr.state, '')) = 'done'
              AND gr.date_done IS NOT NULL
              AND po.date_approve IS NOT NULL
              AND gr.date_done >= ?
              AND gr.date_done < ?
            GROUP BY DATE_TRUNC('month', gr.date_done)::date
        )
        SELECT
            TO_CHAR(months.month, 'Mon') AS month,
            COALESCE(monthly.avg_cycle_days, 0) AS avg_cycle_days
        FROM months
        LEFT JOIN monthly ON monthly.month = months.month
        ORDER BY months.month
    `, start, end, companyID, start, end.AddDate(0, 0, 1)).Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) ProcurementCycleTimeTrend(ctx context.Context, companyID int64, start, end time.Time) ([]trendRow, error) {
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
                DATE_TRUNC('month', gr.date_done)::date AS month,
                ROUND(AVG(EXTRACT(EPOCH FROM (gr.date_done - po.pr_confirm_date::timestamp)) / 86400), 1) AS avg_cycle_days
            FROM procurement.goods_receipts gr
            JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id
            WHERE gr.company_source_id = ?
              AND LOWER(COALESCE(gr.state, '')) = 'done'
              AND gr.date_done IS NOT NULL
              AND po.pr_confirm_date IS NOT NULL
              AND gr.date_done >= ?
              AND gr.date_done < ?
            GROUP BY DATE_TRUNC('month', gr.date_done)::date
        )
        SELECT
            TO_CHAR(months.month, 'Mon') AS month,
            COALESCE(monthly.avg_cycle_days, 0) AS avg_cycle_days
        FROM months
        LEFT JOIN monthly ON monthly.month = months.month
        ORDER BY months.month
    `, start, end, companyID, start, end.AddDate(0, 0, 1)).Scan(&rows).Error
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
			  AND LOWER(COALESCE(state, '')) IN ('purchase', 'done')
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

func (r *gormRepo) ListDocuments(ctx context.Context, companyID int64, start, end time.Time, metricType string, search string, offset, limit int) ([]DocumentRow, int64, error) {
	var rows []DocumentRow
	var total int64
	var err error

	search = strings.TrimSpace(search)

	switch metricType {
	case "pr_no_po":
		q := r.db.WithContext(ctx).Table("procurement.purchase_requests pr").
			Where("pr.company_source_id = ?", companyID).
			Where("pr.date_start BETWEEN ? AND ?", start, end).
			Where("LOWER(COALESCE(pr.state, '')) IN ('purchase_request', 'purchase request')").
			Where(`NOT EXISTS (
				SELECT 1 FROM procurement.purchase_order_purchase_requests rel
				JOIN procurement.purchase_orders po ON po.source_id = rel.purchase_order_source_id
				WHERE rel.purchase_request_source_id = pr.source_id
				  AND po.company_source_id = pr.company_source_id
				  AND LOWER(COALESCE(po.state, '')) <> 'cancel'
			)`).
			Where(`NOT EXISTS (
				SELECT 1 FROM procurement.purchase_orders po
				WHERE po.company_source_id = pr.company_source_id
				  AND po.origin = pr.name
				  AND LOWER(COALESCE(po.state, '')) <> 'cancel'
			)`)

		if search != "" {
			q = q.Where("pr.name ILIKE ?", "%"+search+"%")
		}

		err = q.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		err = q.Select(`
			pr.source_id as id, 
			'PR' as type, 
			pr.name as name, 
			TO_CHAR(pr.date_start, 'YYYY-MM-DD') as date, 
			pr.state as state, 
			pr.amount_total as amount,
			pr.department_source_id as department_source_id,
			pr.branch_source_id as branch_source_id
		`).
			Order("pr.date_start DESC").
			Offset(offset).Limit(limit).Scan(&rows).Error

	case "po_incomplete":
		q := r.db.WithContext(ctx).Table("procurement.purchase_orders po").
			Where("po.company_source_id = ?", companyID).
			Where("po.date_order BETWEEN ? AND ?", start, end).
			Where(`LOWER(COALESCE(po.state, '')) IN (
                  'request_for_amendment', 'rfq', 'draft', 'rfq_sent', 'sent', 
                  'to_approve', 'waiting_for_approve', 'waiting_for_approval', 
                  'waiting_approval', 'rfq_approved', 'over_budget_approval', 
                  'on_hold', 'blanket_ordered'
			)`)

		if search != "" {
			q = q.Where(poNameILike, "%"+search+"%")
		}

		err = q.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		err = q.Select(`
			po.source_id as id, 
			'PO' as type, 
			po.name as name, 
			TO_CHAR(po.date_order, 'YYYY-MM-DD') as date, 
			po.state as state, 
			po.amount_total as amount,
			po.branch_source_id as branch_source_id,
			po.partner_source_id as partner_source_id,
			TO_CHAR(po.date_approve, 'YYYY-MM-DD') as date_approve,
			TO_CHAR(po.date_planned, 'YYYY-MM-DD') as date_planned,
			po.cycle_days as cycle_days,
			po.amount_untaxed as amount_untaxed,
			COALESCE(
				COALESCE(
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_order_purchase_requests rel
						JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
						WHERE rel.purchase_order_source_id = po.source_id
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					),
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_requests pr
						WHERE pr.name = po.origin
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					)
				) - po.amount_untaxed,
				0
			) as amount_saved_from_cost_savings,
			COALESCE(
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_order_purchase_requests rel
					JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
					WHERE rel.purchase_order_source_id = po.source_id
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				),
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_requests pr
					WHERE pr.name = po.origin
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				)
			) as pr_estimated_total,
			po.from_purchase_request as from_purchase_request,
			po.is_goods_orders as is_goods_orders,
			po.product_type as product_type,
			TO_CHAR(po.pr_confirm_date, 'YYYY-MM-DD') as pr_confirm_date,
			po.origin as origin
		`).
			Order(poDateOrderDesc).
			Offset(offset).Limit(limit).Scan(&rows).Error

	case "po_no_receiving":
		q := r.db.WithContext(ctx).Table("procurement.purchase_orders po").
			Where("po.company_source_id = ?", companyID).
			Where("po.date_order BETWEEN ? AND ?", start, end).
			Where("LOWER(COALESCE(po.state, '')) IN ('purchase', 'blanket_ordered')").
			Where(`NOT EXISTS (
				SELECT 1 FROM procurement.goods_receipts gr
				WHERE gr.company_source_id = po.company_source_id
				  AND gr.po_source_id = po.source_id
				  AND LOWER(COALESCE(gr.state, '')) = 'done'
			)`)

		if search != "" {
			q = q.Where(poNameILike, "%"+search+"%")
		}

		err = q.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		err = q.Select(`
			po.source_id as id, 
			'PO' as type, 
			po.name as name, 
			TO_CHAR(po.date_order, 'YYYY-MM-DD') as date, 
			po.state as state, 
			po.amount_total as amount,
			po.branch_source_id as branch_source_id,
			po.partner_source_id as partner_source_id,
			TO_CHAR(po.date_approve, 'YYYY-MM-DD') as date_approve,
			TO_CHAR(po.date_planned, 'YYYY-MM-DD') as date_planned,
			po.cycle_days as cycle_days,
			po.amount_untaxed as amount_untaxed,
			COALESCE(
				COALESCE(
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_order_purchase_requests rel
						JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
						WHERE rel.purchase_order_source_id = po.source_id
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					),
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_requests pr
						WHERE pr.name = po.origin
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					)
				) - po.amount_untaxed,
				0
			) as amount_saved_from_cost_savings,
			COALESCE(
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_order_purchase_requests rel
					JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
					WHERE rel.purchase_order_source_id = po.source_id
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				),
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_requests pr
					WHERE pr.name = po.origin
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				)
			) as pr_estimated_total,
			po.from_purchase_request as from_purchase_request,
			po.is_goods_orders as is_goods_orders,
			po.product_type as product_type,
			TO_CHAR(po.pr_confirm_date, 'YYYY-MM-DD') as pr_confirm_date,
			po.origin as origin
		`).
			Order(poDateOrderDesc).
			Offset(offset).Limit(limit).Scan(&rows).Error

	case "cost_saving", "saving_rate":
		q := r.db.WithContext(ctx).Table("procurement.purchase_orders po").
			Where("po.company_source_id = ?", companyID).
			Where("po.date_order BETWEEN ? AND ?", start, end).
			Where("po.from_purchase_request = true").
			Where("LOWER(COALESCE(po.state, '')) IN ('purchase', 'done')")

		if search != "" {
			q = q.Where(poNameILike, "%"+search+"%")
		}

		err = q.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		err = q.Select(`
			po.source_id as id, 
			'PO' as type, 
			po.name as name, 
			TO_CHAR(po.date_order, 'YYYY-MM-DD') as date, 
			po.state as state, 
			po.amount_total as amount,
			po.branch_source_id as branch_source_id,
			po.partner_source_id as partner_source_id,
			TO_CHAR(po.date_approve, 'YYYY-MM-DD') as date_approve,
			TO_CHAR(po.date_planned, 'YYYY-MM-DD') as date_planned,
			po.cycle_days as cycle_days,
			po.amount_untaxed as amount_untaxed,
			COALESCE(
				COALESCE(
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_order_purchase_requests rel
						JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
						WHERE rel.purchase_order_source_id = po.source_id
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					),
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_requests pr
						WHERE pr.name = po.origin
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					)
				) - po.amount_untaxed,
				0
			) as amount_saved_from_cost_savings,
			COALESCE(
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_order_purchase_requests rel
					JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
					WHERE rel.purchase_order_source_id = po.source_id
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				),
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_requests pr
					WHERE pr.name = po.origin
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				)
			) as pr_estimated_total,
			po.from_purchase_request as from_purchase_request,
			po.is_goods_orders as is_goods_orders,
			po.product_type as product_type,
			TO_CHAR(po.pr_confirm_date, 'YYYY-MM-DD') as pr_confirm_date,
			po.origin as origin
		`).
			Order(poDateOrderDesc).
			Offset(offset).Limit(limit).Scan(&rows).Error

	case "po_cycle_time":
		q := r.db.WithContext(ctx).Table("procurement.purchase_orders po").
			Where("po.company_source_id = ?", companyID).
			Where("po.date_approve BETWEEN ? AND ?", start, end).
			Where("po.from_purchase_request = true").
			Where("LOWER(COALESCE(po.state, '')) IN ('purchase', 'done')")

		if search != "" {
			q = q.Where(poNameILike, "%"+search+"%")
		}

		err = q.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		err = q.Select(`
			po.source_id as id, 
			'PO' as type, 
			po.name as name, 
			TO_CHAR(po.date_order, 'YYYY-MM-DD') as date, 
			po.state as state, 
			po.amount_total as amount,
			po.branch_source_id as branch_source_id,
			po.partner_source_id as partner_source_id,
			TO_CHAR(po.date_approve, 'YYYY-MM-DD') as date_approve,
			TO_CHAR(po.date_planned, 'YYYY-MM-DD') as date_planned,
			po.cycle_days as cycle_days,
			po.amount_untaxed as amount_untaxed,
			COALESCE(
				COALESCE(
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_order_purchase_requests rel
						JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
						WHERE rel.purchase_order_source_id = po.source_id
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					),
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_requests pr
						WHERE pr.name = po.origin
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					)
				) - po.amount_untaxed,
				0
			) as amount_saved_from_cost_savings,
			COALESCE(
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_order_purchase_requests rel
					JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
					WHERE rel.purchase_order_source_id = po.source_id
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				),
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_requests pr
					WHERE pr.name = po.origin
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				)
			) as pr_estimated_total,
			po.from_purchase_request as from_purchase_request,
			po.is_goods_orders as is_goods_orders,
			po.product_type as product_type,
			TO_CHAR(po.pr_confirm_date, 'YYYY-MM-DD') as pr_confirm_date,
			po.origin as origin
		`).
			Order("po.date_approve DESC").
			Offset(offset).Limit(limit).Scan(&rows).Error

	case "otd_rate", "receiving_cycle_time":
		q := r.db.WithContext(ctx).Table("procurement.goods_receipts gr").
			Joins("JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id").
			Where("gr.company_source_id = ?", companyID).
			Where("LOWER(COALESCE(gr.state, '')) = 'done'").
			Where("gr.date_done IS NOT NULL AND po.date_approve IS NOT NULL").
			Where("gr.date_done >= ? AND gr.date_done < ?", start, end.AddDate(0, 0, 1))

		if metricType == "otd_rate" {
			q = q.Where("NOT (COALESCE(po.is_goods_orders, false) = true AND COALESCE(po.product_type, '') = 'storable')")
		}

		if search != "" {
			q = q.Where(poNameILike, "%"+search+"%")
		}

		err = q.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		err = q.Select(`
			po.source_id as id, 
			'PO' as type, 
			po.name as name, 
			TO_CHAR(po.date_order, 'YYYY-MM-DD') as date, 
			po.state as state, 
			po.amount_total as amount,
			po.branch_source_id as branch_source_id,
			po.partner_source_id as partner_source_id,
			TO_CHAR(po.date_approve, 'YYYY-MM-DD') as date_approve,
			TO_CHAR(po.date_planned, 'YYYY-MM-DD') as date_planned,
			ROUND((EXTRACT(EPOCH FROM (gr.date_done - po.date_approve::timestamp)) / 86400)::numeric, 1) as cycle_days,
			po.amount_untaxed as amount_untaxed,
			COALESCE(
				COALESCE(
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_order_purchase_requests rel
						JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
						WHERE rel.purchase_order_source_id = po.source_id
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					),
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_requests pr
						WHERE pr.name = po.origin
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					)
				) - po.amount_untaxed,
				0
			) as amount_saved_from_cost_savings,
			COALESCE(
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_order_purchase_requests rel
					JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
					WHERE rel.purchase_order_source_id = po.source_id
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				),
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_requests pr
					WHERE pr.name = po.origin
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				)
			) as pr_estimated_total,
			po.from_purchase_request as from_purchase_request,
			po.is_goods_orders as is_goods_orders,
			po.product_type as product_type,
			TO_CHAR(po.pr_confirm_date, 'YYYY-MM-DD') as pr_confirm_date,
			po.origin as origin
		`).
			Order("gr.date_done DESC").
			Offset(offset).Limit(limit).Scan(&rows).Error

	case "procurement_cycle_time":
		q := r.db.WithContext(ctx).Table("procurement.goods_receipts gr").
			Joins("JOIN procurement.purchase_orders po ON po.source_id = gr.po_source_id").
			Where("gr.company_source_id = ?", companyID).
			Where("LOWER(COALESCE(gr.state, '')) = 'done'").
			Where("gr.date_done IS NOT NULL AND po.pr_confirm_date IS NOT NULL").
			Where("gr.date_done >= ? AND gr.date_done < ?", start, end.AddDate(0, 0, 1))

		if search != "" {
			q = q.Where(poNameILike, "%"+search+"%")
		}

		err = q.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}
		err = q.Select(`
			po.source_id as id, 
			'PO' as type, 
			po.name as name, 
			TO_CHAR(po.date_order, 'YYYY-MM-DD') as date, 
			po.state as state, 
			po.amount_total as amount,
			po.branch_source_id as branch_source_id,
			po.partner_source_id as partner_source_id,
			TO_CHAR(po.date_approve, 'YYYY-MM-DD') as date_approve,
			TO_CHAR(po.date_planned, 'YYYY-MM-DD') as date_planned,
			ROUND((EXTRACT(EPOCH FROM (gr.date_done - po.pr_confirm_date::timestamp)) / 86400)::numeric, 1) as cycle_days,
			po.amount_untaxed as amount_untaxed,
			COALESCE(
				COALESCE(
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_order_purchase_requests rel
						JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
						WHERE rel.purchase_order_source_id = po.source_id
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					),
					(
						SELECT SUM(pr.amount_total)
						FROM procurement.purchase_requests pr
						WHERE pr.name = po.origin
						  AND pr.company_source_id = po.company_source_id
						  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
					)
				) - po.amount_untaxed,
				0
			) as amount_saved_from_cost_savings,
			COALESCE(
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_order_purchase_requests rel
					JOIN procurement.purchase_requests pr ON pr.source_id = rel.purchase_request_source_id
					WHERE rel.purchase_order_source_id = po.source_id
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				),
				(
					SELECT SUM(pr.amount_total)
					FROM procurement.purchase_requests pr
					WHERE pr.name = po.origin
					  AND pr.company_source_id = po.company_source_id
					  AND LOWER(COALESCE(pr.state, '')) NOT IN ('cancel', 'rejected')
				)
			) as pr_estimated_total,
			po.from_purchase_request as from_purchase_request,
			po.is_goods_orders as is_goods_orders,
			po.product_type as product_type,
			TO_CHAR(po.pr_confirm_date, 'YYYY-MM-DD') as pr_confirm_date,
			po.origin as origin
		`).
			Order("gr.date_done DESC").
			Offset(offset).Limit(limit).Scan(&rows).Error

	default:
		return []DocumentRow{}, 0, nil
	}

	return rows, total, err
}
