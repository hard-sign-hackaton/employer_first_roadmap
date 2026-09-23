package services

import (
	"context"
	"efr_bot/dto"
	"efr_bot/services/ports"
)

type ReferenceService interface {
	ListRegions(context.Context) ([]dto.RegionResponse, error)
	ListInterestTags(context.Context) ([]dto.InterestResponse, error)
	ListExamSubjects(context.Context) ([]dto.ExamSubjectResponse, error)
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

func (s *referenceService) ListInterestTags(ctx context.Context) ([]dto.InterestResponse, error) {
	values, err := s.store.ListInterestTags(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.InterestResponse, 0, len(values))
	for _, value := range values {
		result = append(result, dto.InterestResponse{InterestTagID: value.ID, Name: value.Name})
	}
	return result, nil
}

func (s *referenceService) ListExamSubjects(ctx context.Context) ([]dto.ExamSubjectResponse, error) {
	values, err := s.store.ListExamSubjects(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ExamSubjectResponse, 0, len(values))
	for _, value := range values {
		result = append(result, dto.ExamSubjectResponse{ID: value.ID, Name: value.Name})
	}
	return result, nil
}
