package services

import (
	"context"
	"efr_bot/dto"
	"efr_bot/services/ports"
)

type ReferenceService interface {
	ListRegions(context.Context) ([]dto.RegionResponse, error)
}
type referenceService struct{ store ports.ReferenceStore }

func NewReferenceService(store ports.ReferenceStore) ReferenceService {
	return &referenceService{store}
}
func (s *referenceService) ListRegions(ctx context.Context) ([]dto.RegionResponse, error) {
	values, err := s.store.ListRegions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.RegionResponse, 0, len(values))
	for _, v := range values {
		result = append(result, dto.RegionResponse{ID: v.ID, Name: v.Name})
	}
	return result, nil
}
