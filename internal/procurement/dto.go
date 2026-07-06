package procurement

type Metric struct {
	Title   string  `json:"title"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	Remarks string  `json:"remarks,omitempty"`
}

type DashboardResponse struct {
	CompanyID   int64      `json:"company_id"`
	Period      PeriodInfo `json:"period"`
	CostSaving  Metric     `json:"cost_saving"`
	SavingRate  Metric     `json:"saving_rate"`
	OTDRate     Metric     `json:"otd_rate"`
	POCycleTime Metric     `json:"po_cycle_time"`
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
