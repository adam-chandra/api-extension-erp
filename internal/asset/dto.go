package asset

// DashboardResponse is returned by GET /api/asset/dashboard.
type DashboardResponse struct {
	CompanyID       int64       `json:"companyId"`
	TotalUnitAsset  int64       `json:"totalUnitAsset"`
	TotalAssetValue float64     `json:"totalAssetValue"`
	Status          StatusTiles `json:"status"`
}

type StatusTiles struct {
	Active  int64 `json:"active"`
	Idle    int64 `json:"idle"`
	Damage  int64 `json:"damage"`
	Missing int64 `json:"missing"`
}
