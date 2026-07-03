package asset

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	DashboardTiles(ctx context.Context, companyID int64) (dashboardRow, error)
}

type dashboardRow struct {
	TotalUnitAsset  int64   `gorm:"column:total_unit_asset"`
	TotalAssetValue float64 `gorm:"column:total_asset_value"`
	ActiveAsset     int64   `gorm:"column:active_asset"`
	IdleAsset       int64   `gorm:"column:idle_asset"`
	DamageAsset     int64   `gorm:"column:damage_asset"`
	MissingAsset    int64   `gorm:"column:missing_asset"`
}

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) DashboardTiles(ctx context.Context, companyID int64) (dashboardRow, error) {
	var row dashboardRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT
		            COUNT(*) FILTER (
		                WHERE status_normalized IN ('operative', 'breakdown', 'maintenance')
		            ) AS total_unit_asset,
		            COALESCE(SUM(asset_value) FILTER (
		                WHERE status_normalized IN ('operative', 'breakdown', 'maintenance')
		            ), 0)::NUMERIC(20,2) AS total_asset_value,
		            COUNT(*) FILTER (
		                WHERE status_normalized = 'operative'
		                  AND employee_heldby IS NOT NULL
		                  AND BTRIM(employee_heldby) <> ''
		            ) AS active_asset,
		            COUNT(*) FILTER (
		                WHERE status_normalized = 'operative'
		                  AND (employee_heldby IS NULL OR BTRIM(employee_heldby) = '')
		            ) AS idle_asset,
		            COUNT(*) FILTER (WHERE status_normalized = 'breakdown') AS damage_asset,
		            COUNT(*) FILTER (WHERE status_normalized = 'missing')   AS missing_asset
		     FROM asset.asset_equipment
		     WHERE company_source_id = ?`, companyID).
		Scan(&row).Error
	return row, err
}
