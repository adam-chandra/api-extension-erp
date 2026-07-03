package asset

import (
	"context"
	"fmt"
)

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service { return &Service{repo: r} }

func (s *Service) Dashboard(ctx context.Context, companyID int64) (*DashboardResponse, error) {
	row, err := s.repo.DashboardTiles(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("asset dashboard: %w", err)
	}

	return &DashboardResponse{
		CompanyID:       companyID,
		TotalUnitAsset:  row.TotalUnitAsset,
		TotalAssetValue: row.TotalAssetValue,
		Status: StatusTiles{
			Active:  row.ActiveAsset,
			Idle:    row.IdleAsset,
			Damage:  row.DamageAsset,
			Missing: row.MissingAsset,
		},
	}, nil
}
