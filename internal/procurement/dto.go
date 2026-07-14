package procurement

type Metric struct {
	Title   string  `json:"title"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	Remarks string  `json:"remarks,omitempty"`
}

type DashboardResponse struct {
	CompanyID            int64      `json:"company_id"`
	Period               PeriodInfo `json:"period"`
	CostSaving           Metric     `json:"cost_saving"`
	SavingRate           Metric     `json:"saving_rate"`
	OTDRate              Metric     `json:"otd_rate"`
	POCycleTime          Metric     `json:"po_cycle_time"`
	PRNoPO               Metric     `json:"pr_no_po"`
	POIncomplete         Metric     `json:"po_incomplete"`
	PONoReceiving        Metric     `json:"po_no_receiving"`
	ReceivingCycleTime   Metric     `json:"receiving_cycle_time"`
	ProcurementCycleTime Metric     `json:"procurement_cycle_time"`
}

type PeriodInfo struct {
	Code  string `json:"code"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type TrendPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type YTDPoint struct {
	Label       string  `json:"label"`
	YTDThisYear float64 `json:"ytd_this_year"`
	YTDLastYear float64 `json:"ytd_last_year"`
}

type DocumentRow struct {
	ID                         int64    `json:"id"`
	Type                       string   `json:"type"`
	Name                       string   `json:"name"`
	Date                       string   `json:"date"`
	State                      string   `json:"state"`
	Amount                     float64  `json:"amount"`
	DepartmentSourceID         *int64   `json:"department_source_id,omitempty"`
	BranchSourceID             *int64   `json:"branch_source_id,omitempty"`
	PartnerSourceID            *int64   `json:"partner_source_id,omitempty"`
	DateApprove                *string  `json:"date_approve,omitempty"`
	DatePlanned                *string  `json:"date_planned,omitempty"`
	CycleDays                  *float64 `json:"cycle_days,omitempty"`
	AmountUntaxed              *float64 `json:"amount_untaxed,omitempty"`
	AmountSavedFromCostSavings *float64 `json:"amount_saved_from_cost_savings,omitempty"`
	PREstimatedTotal           *float64 `json:"pr_estimated_total,omitempty" gorm:"column:pr_estimated_total"`
	FromPurchaseRequest        *bool    `json:"from_purchase_request,omitempty"`
	IsGoodsOrders              *bool    `json:"is_goods_orders,omitempty"`
	ProductType                *string  `json:"product_type,omitempty"`
	PrConfirmDate              *string  `json:"pr_confirm_date,omitempty"`
	Origin                     *string  `json:"origin,omitempty"`
}

type PaginatedDocuments struct {
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Items []DocumentRow `json:"items"`
}

type ListDocumentsQuery struct {
	PeriodCode  string
	CustomStart string
	CustomEnd   string
	MetricType  string
	Search      string
	Page        int
	Limit       int
}
