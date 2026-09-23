package services

import (
	"context"
	"fmt"

	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/services/ports"
)

type profileService struct{ store ports.ProfileStore }

var _ ProfileService = (*profileService)(nil)

func NewProfileService(store ports.ProfileStore) ProfileService { return &profileService{store: store} }

func (s *profileService) SaveProfile(ctx context.Context, userID int64, request dto.UpsertProfileRequest) (dto.ProfileResponse, error) {
	if request.Grade < 9 || request.Grade > 11 || request.RegionID <= 0 {
		return dto.ProfileResponse{}, fmt.Errorf("invalid profile")
	}
	profile, err := s.store.SaveProfile(ctx, models.UserProfile{ID: userID, Grade: request.Grade, RegionID: request.RegionID, WillingToRelocate: request.WillingToRelocate})
	if err != nil {
		return dto.ProfileResponse{}, err
	}
	return dto.ProfileResponse{UserID: profile.ID, Grade: profile.Grade, Region: dto.RegionResponse{ID: profile.Region.ID, Name: profile.Region.Name}, WillingToRelocate: profile.WillingToRelocate}, nil
}
func (s *profileService) GetProfile(ctx context.Context, userID int64) (dto.ProfileResponse, error) {
	p, err := s.store.FindProfileByUserID(ctx, userID)
	return dto.ProfileResponse{UserID: p.ID, Grade: p.Grade, Region: dto.RegionResponse{ID: p.Region.ID, Name: p.Region.Name}, WillingToRelocate: p.WillingToRelocate}, err
}
func (s *profileService) SaveSurveyInterests(ctx context.Context, userID int64, request dto.SaveSurveyInterestsRequest) ([]dto.InterestResponse, error) {
	return nil, fmt.Errorf("full survey is not implemented")
}
func (s *profileService) SaveUserSubjects(ctx context.Context, userID int64, request dto.SaveUserSubjectsRequest) ([]dto.UserSubjectResponse, error) {
	items := make([]models.UserSubject, 0, len(request.Subjects))
	for _, v := range request.Subjects {
		if v.ExamSubjectID <= 0 || (v.Status != models.SubjectStatusPlanned && v.Status != models.SubjectStatusSelected) {
			return nil, fmt.Errorf("invalid subject")
		}
		items = append(items, models.UserSubject{ExamSubjectID: v.ExamSubjectID, Status: v.Status, ExpectedScore: v.ExpectedScore})
	}
	items, err := s.store.ReplaceUserSubjects(ctx, userID, items)
	if err != nil {
		return nil, err
	}
	return subjectResponses(items), nil
}
func (s *profileService) SaveExamResults(context.Context, int64, dto.SaveExamResultsRequest) ([]dto.UserSubjectResponse, error) {
	return nil, fmt.Errorf("exam results are not implemented")
}
