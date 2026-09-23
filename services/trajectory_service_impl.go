package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/services/ports"
)

// trajectoryService реализует сценарий выбора работодателя, карьерного направления и ЕГЭ.
type trajectoryService struct {
	careers   ports.CareerStore
	education ports.EducationStore
	profiles  ports.ProfileStore
	roadmaps  ports.RoadmapStore
}

var _ TrajectoryService = (*trajectoryService)(nil)

// NewTrajectoryService создаёт сервис выбора работодателя и карьерного направления.
func NewTrajectoryService(
	careers ports.CareerStore,
	education ports.EducationStore,
	profiles ports.ProfileStore,
	roadmaps ports.RoadmapStore,
) TrajectoryService {
	return &trajectoryService{
		careers:   careers,
		education: education,
		profiles:  profiles,
		roadmaps:  roadmaps,
	}
}

func (s *trajectoryService) FindCompanies(ctx context.Context, request dto.FindCompaniesRequest) ([]dto.CompanyCatalogItemResponse, error) {
	if request.Limit < 0 {
		return nil, fmt.Errorf("limit must not be negative")
	}

	companies, err := s.careers.FindCompanies(ctx, strings.TrimSpace(request.Query), request.Limit)
	if err != nil {
		return nil, fmt.Errorf("find companies: %w", err)
	}

	result := make([]dto.CompanyCatalogItemResponse, 0, len(companies))
	for _, company := range companies {
		result = append(result, companyResponse(company))
	}
	return result, nil
}

func (s *trajectoryService) RecommendCompanies(ctx context.Context, userID int64, request dto.RecommendCompaniesRequest) ([]dto.RecommendedCompanyResponse, error) {
	if request.Limit < 0 {
		return nil, fmt.Errorf("limit must not be negative")
	}

	profile, err := s.profiles.FindProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user profile: %w", err)
	}
	interests, err := s.profiles.ListUserInterests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user interests: %w", err)
	}
	companies, err := s.careers.ListCompaniesWithDirectionsAndTags(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("list companies for recommendation: %w", err)
	}

	var selectedSubjectIDs []int64
	if profile.Grade == 11 {
		selectedSubjectIDs, err = s.selectedSubjectIDs(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	userWeights := interestWeights(interests)
	result := make([]dto.RecommendedCompanyResponse, 0, len(companies))

	for _, company := range companies {
		score, reasons := bestDirectionScore(company.CareerDirections, userWeights)
		compatible := true
		if profile.Grade == 11 && len(selectedSubjectIDs) > 0 {
			directionIDs, listErr := s.education.ListDirectionIDsAvailableForSubjects(
				ctx,
				company.ID,
				selectedSubjectIDs,
				targetAdmissionYear(profile.Grade, time.Now()),
			)
			if listErr != nil {
				return nil, fmt.Errorf("check company %d exam compatibility: %w", company.ID, listErr)
			}
			compatible = len(directionIDs) > 0
			if compatible {
				reasons = append(reasons, "Есть направление с выбранным набором ЕГЭ")
			} else {
				reasons = append(reasons, "Нет направления с выбранным набором ЕГЭ")
			}
		}

		result = append(result, dto.RecommendedCompanyResponse{
			CompanyCatalogItemResponse: companyResponse(company),
			Score:                      score,
			Reasons:                    reasons,
			IsCompatible:               compatible,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].IsCompatible != result[j].IsCompatible {
			return result[i].IsCompatible
		}
		if result[i].Score != result[j].Score {
			return result[i].Score > result[j].Score
		}
		return result[i].Name < result[j].Name
	})
	if request.Limit > 0 && len(result) > request.Limit {
		result = result[:request.Limit]
	}
	return result, nil
}

func (s *trajectoryService) SelectCompany(ctx context.Context, _ int64, request dto.SelectCompanyRequest) (dto.CompanyCatalogItemResponse, error) {
	if request.CompanyID <= 0 {
		return dto.CompanyCatalogItemResponse{}, fmt.Errorf("company_id must be positive")
	}

	company, err := s.careers.FindCompanyByID(ctx, request.CompanyID)
	if err != nil {
		return dto.CompanyCatalogItemResponse{}, fmt.Errorf("find selected company: %w", err)
	}
	return companyResponse(company), nil
}

func (s *trajectoryService) GetCareerDirections(ctx context.Context, userID int64, request dto.GetCareerDirectionsRequest) ([]dto.CareerDirectionResponse, error) {
	if request.CompanyID <= 0 {
		return nil, fmt.Errorf("company_id must be positive")
	}
	if _, err := s.careers.FindCompanyByID(ctx, request.CompanyID); err != nil {
		return nil, fmt.Errorf("find company: %w", err)
	}

	profile, err := s.profiles.FindProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user profile: %w", err)
	}
	interests, err := s.profiles.ListUserInterests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user interests: %w", err)
	}
	directions, err := s.careers.ListCareerDirectionsByCompany(ctx, request.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("list company career directions: %w", err)
	}

	if profile.Grade == 11 {
		selectedSubjectIDs, subjectErr := s.selectedSubjectIDs(ctx, userID)
		if subjectErr != nil {
			return nil, subjectErr
		}
		if len(selectedSubjectIDs) > 0 {
			availableIDs, listErr := s.education.ListDirectionIDsAvailableForSubjects(
				ctx,
				request.CompanyID,
				selectedSubjectIDs,
				targetAdmissionYear(profile.Grade, time.Now()),
			)
			if listErr != nil {
				return nil, fmt.Errorf("list directions available for selected exams: %w", listErr)
			}
			directions = filterDirections(directions, availableIDs)
		}
	}

	userWeights := interestWeights(interests)
	result := make([]dto.CareerDirectionResponse, 0, len(directions))
	for _, direction := range directions {
		score, reasons := directionScore(direction, userWeights)
		response := dto.CareerDirectionResponse{
			ID:          direction.ID,
			CompanyID:   direction.CompanyID,
			Name:        direction.Name,
			Description: direction.Description,
			Reasons:     reasons,
		}
		if len(userWeights) > 0 {
			response.Score = &score
		}
		result = append(result, response)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score != nil && result[j].Score != nil && *result[i].Score != *result[j].Score {
			return *result[i].Score > *result[j].Score
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *trajectoryService) GetRecommendedExamSets(ctx context.Context, request dto.GetRecommendedExamSetsRequest) ([]dto.RecommendedExamSetResponse, error) {
	if request.CareerDirectionID <= 0 {
		return nil, fmt.Errorf("career_direction_id must be positive")
	}
	if request.AdmissionYear < 2020 || request.AdmissionYear > 2100 {
		return nil, fmt.Errorf("admission_year must be between 2020 and 2100")
	}

	if _, err := s.careers.FindCareerDirectionByID(ctx, request.CareerDirectionID); err != nil {
		return nil, fmt.Errorf("find career direction: %w", err)
	}
	combinations, err := s.education.ListExamCombinationsForDirection(ctx, request.CareerDirectionID, request.AdmissionYear)
	if err != nil {
		return nil, fmt.Errorf("list exam combinations: %w", err)
	}

	type examSet struct {
		ids         []int64
		subjects    []dto.ExamSubjectResponse
		description string
	}
	uniqueSets := make(map[string]examSet)
	for _, combination := range combinations {
		ids := make([]int64, 0, len(combination.Items))
		subjects := make([]dto.ExamSubjectResponse, 0, len(combination.Items))
		for _, item := range combination.Items {
			ids = append(ids, item.ExamSubjectID)
			subjects = append(subjects, dto.ExamSubjectResponse{ID: item.ExamSubjectID, Name: item.ExamSubject.Name})
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		sort.Slice(subjects, func(i, j int) bool { return subjects[i].ID < subjects[j].ID })
		key := examSetKey(ids)
		if _, exists := uniqueSets[key]; exists {
			continue
		}
		uniqueSets[key] = examSet{
			ids:         ids,
			subjects:    subjects,
			description: examSetDescription(combination),
		}
	}

	keys := make([]string, 0, len(uniqueSets))
	for key := range uniqueSets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]dto.RecommendedExamSetResponse, 0, len(keys))
	for _, key := range keys {
		set := uniqueSets[key]
		result = append(result, dto.RecommendedExamSetResponse{
			ExamSubjectIDs: set.ids,
			Subjects:       set.subjects,
			Description:    set.description,
		})
	}
	return result, nil
}

func (s *trajectoryService) ConfirmGoal(ctx context.Context, userID int64, request dto.ConfirmGoalRequest) (dto.GoalResponse, error) {
	if request.CareerDirectionID <= 0 {
		return dto.GoalResponse{}, fmt.Errorf("career_direction_id must be positive")
	}
	if request.TargetAdmissionYear < 2020 || request.TargetAdmissionYear > 2100 {
		return dto.GoalResponse{}, fmt.Errorf("target_admission_year must be between 2020 and 2100")
	}
	if _, err := s.profiles.FindProfileByUserID(ctx, userID); err != nil {
		return dto.GoalResponse{}, fmt.Errorf("find user profile: %w", err)
	}

	direction, err := s.careers.FindCareerDirectionByID(ctx, request.CareerDirectionID)
	if err != nil {
		return dto.GoalResponse{}, fmt.Errorf("find career direction: %w", err)
	}
	goal, err := s.roadmaps.CreateGoal(ctx, models.UserGoal{
		UserProfileID:       userID,
		CareerDirectionID:   direction.ID,
		TargetAdmissionYear: request.TargetAdmissionYear,
		Status:              models.GoalStatusActive,
		CreatedAt:           time.Now().UTC(),
	})
	if err != nil {
		return dto.GoalResponse{}, fmt.Errorf("create user goal: %w", err)
	}

	subjects, err := s.profiles.ListUserSubjects(ctx, userID)
	if err != nil {
		return dto.GoalResponse{}, fmt.Errorf("list user subjects: %w", err)
	}
	return dto.GoalResponse{
		ID:      goal.ID,
		Company: companyResponse(direction.Company),
		CareerDirection: dto.CareerDirectionResponse{
			ID:          direction.ID,
			CompanyID:   direction.CompanyID,
			Name:        direction.Name,
			Description: direction.Description,
		},
		TargetAdmissionYear: goal.TargetAdmissionYear,
		Status:              goal.Status,
		Subjects:            subjectResponses(subjects),
	}, nil
}

func (s *trajectoryService) selectedSubjectIDs(ctx context.Context, userID int64) ([]int64, error) {
	subjects, err := s.profiles.ListUserSubjects(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user subjects: %w", err)
	}

	ids := make([]int64, 0, len(subjects))
	for _, subject := range subjects {
		if subject.Status == models.SubjectStatusSelected || subject.Status == models.SubjectStatusPassed {
			ids = append(ids, subject.ExamSubjectID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func companyResponse(company models.Company) dto.CompanyCatalogItemResponse {
	return dto.CompanyCatalogItemResponse{
		ID:          company.ID,
		Name:        company.Name,
		Description: company.Description,
	}
}

func interestWeights(interests []models.UserInterest) map[int64]float64 {
	weights := make(map[int64]float64, len(interests))
	for _, interest := range interests {
		weights[interest.InterestTagID] = interest.Weight
	}
	return weights
}

func bestDirectionScore(directions []models.CareerDirection, userWeights map[int64]float64) (float64, []string) {
	if len(userWeights) == 0 {
		return 0, nil
	}

	bestScore := 0.0
	var bestReasons []string
	for _, direction := range directions {
		score, reasons := directionScore(direction, userWeights)
		if score > bestScore {
			bestScore = score
			bestReasons = append([]string{"Подходит направление «" + direction.Name + "»"}, reasons...)
		}
	}
	return bestScore, bestReasons
}

func directionScore(direction models.CareerDirection, userWeights map[int64]float64) (float64, []string) {
	if len(userWeights) == 0 || len(direction.InterestTags) == 0 {
		return 0, nil
	}

	userNorm := 0.0
	for _, weight := range userWeights {
		userNorm += weight * weight
	}
	directionNorm := 0.0
	dotProduct := 0.0
	type match struct {
		name         string
		contribution float64
	}
	matches := make([]match, 0)
	for _, tag := range direction.InterestTags {
		directionNorm += tag.Weight * tag.Weight
		if userWeight, exists := userWeights[tag.InterestTagID]; exists {
			contribution := userWeight * tag.Weight
			dotProduct += contribution
			matches = append(matches, match{name: tag.InterestTag.Name, contribution: contribution})
		}
	}
	if userNorm == 0 || directionNorm == 0 {
		return 0, nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].contribution != matches[j].contribution {
			return matches[i].contribution > matches[j].contribution
		}
		return matches[i].name < matches[j].name
	})
	reasons := make([]string, 0, len(matches))
	for _, matched := range matches {
		reasons = append(reasons, "Совпадает интерес «"+matched.name+"»")
	}
	return dotProduct / (math.Sqrt(userNorm) * math.Sqrt(directionNorm)), reasons
}

func filterDirections(directions []models.CareerDirection, availableIDs []int64) []models.CareerDirection {
	available := make(map[int64]struct{}, len(availableIDs))
	for _, id := range availableIDs {
		available[id] = struct{}{}
	}

	result := make([]models.CareerDirection, 0, len(directions))
	for _, direction := range directions {
		if _, exists := available[direction.ID]; exists {
			result = append(result, direction)
		}
	}
	return result
}

func targetAdmissionYear(grade int16, now time.Time) int16 {
	year := now.Year()
	if now.Month() >= time.September {
		year++
	}
	year += int(11 - grade)
	return int16(year)
}

func examSetKey(ids []int64) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return strings.Join(parts, ",")
}

func examSetDescription(combination models.ExamCombination) string {
	program := combination.EducationProgram
	if program.Name == "" {
		return ""
	}
	if program.University.Name == "" {
		return "Подходит для программы «" + program.Name + "»"
	}
	return "Подходит для программы «" + program.Name + "» в «" + program.University.Name + "»"
}

func subjectResponses(subjects []models.UserSubject) []dto.UserSubjectResponse {
	result := make([]dto.UserSubjectResponse, 0, len(subjects))
	for _, subject := range subjects {
		result = append(result, dto.UserSubjectResponse{
			ExamSubjectID: subject.ExamSubjectID,
			Name:          subject.ExamSubject.Name,
			Status:        subject.Status,
			ExpectedScore: subject.ExpectedScore,
			ActualScore:   subject.ActualScore,
		})
	}
	return result
}
