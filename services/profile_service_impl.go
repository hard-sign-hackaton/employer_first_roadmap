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
	if _, err := s.store.FindProfileByUserID(ctx, userID); err != nil {
		return nil, fmt.Errorf("find user profile: %w", err)
	}
	interests := make([]models.UserInterest, 0, len(request.Interests))
	seen := make(map[int64]struct{}, len(request.Interests))
	for _, item := range request.Interests {
		if item.InterestTagID <= 0 || item.Weight < 0 {
			return nil, fmt.Errorf("invalid interest")
		}
		if _, exists := seen[item.InterestTagID]; exists {
			return nil, fmt.Errorf("interest tag %d is duplicated", item.InterestTagID)
		}
		seen[item.InterestTagID] = struct{}{}
		interests = append(interests, models.UserInterest{InterestTagID: item.InterestTagID, Weight: item.Weight})
	}
	saved, err := s.store.ReplaceUserInterests(ctx, userID, interests)
	if err != nil {
		return nil, err
	}
	result := make([]dto.InterestResponse, 0, len(saved))
	for _, item := range saved {
		result = append(result, dto.InterestResponse{InterestTagID: item.InterestTagID, Name: item.InterestTag.Name, Weight: item.Weight})
	}
	return result, nil
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
func (s *profileService) GetUserSubjects(ctx context.Context, userID int64) ([]dto.UserSubjectResponse, error) {
	items, err := s.store.ListUserSubjects(ctx, userID)
	if err != nil {
		return nil, err
	}
	return subjectResponses(items), nil
}
func (s *profileService) SaveExamResults(ctx context.Context, userID int64, request dto.SaveExamResultsRequest) ([]dto.UserSubjectResponse, error) {
	if len(request.Results) == 0 {
		return nil, fmt.Errorf("no exam results provided")
	}
	items := make([]models.UserSubject, 0, len(request.Results))
	seen := make(map[int64]struct{}, len(request.Results))
	for _, result := range request.Results {
		if result.ExamSubjectID <= 0 || result.ActualScore < 0 || result.ActualScore > 100 {
			return nil, fmt.Errorf("invalid exam result")
		}
		if _, exists := seen[result.ExamSubjectID]; exists {
			return nil, fmt.Errorf("exam subject %d is duplicated", result.ExamSubjectID)
		}
		seen[result.ExamSubjectID] = struct{}{}
		score := result.ActualScore
		items = append(items, models.UserSubject{
			ExamSubjectID: result.ExamSubjectID,
			Status:        models.SubjectStatusPassed,
			ActualScore:   &score,
		})
	}
	saved, err := s.store.SaveExamResults(ctx, userID, items)
	if err != nil {
		return nil, err
	}
	return subjectResponses(saved), nil
}
