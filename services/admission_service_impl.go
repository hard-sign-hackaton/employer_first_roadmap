package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/services/ports"
	"gorm.io/gorm"
)

// admissionService корректирует roadmap после результатов ЕГЭ и приёмной кампании.
type admissionService struct {
	education ports.EducationStore
	profiles  ports.ProfileStore
	roadmaps  ports.RoadmapStore
	careers   ports.CareerStore
}

var _ AdmissionService = (*admissionService)(nil)

// NewAdmissionService создаёт сервис подбора программ и фиксации результатов поступления.
func NewAdmissionService(
	education ports.EducationStore,
	profiles ports.ProfileStore,
	roadmaps ports.RoadmapStore,
	careers ports.CareerStore,
) AdmissionService {
	return &admissionService{
		education: education,
		profiles:  profiles,
		roadmaps:  roadmaps,
		careers:   careers,
	}
}

func (s *admissionService) FindEducationOptions(ctx context.Context, userID int64, request dto.FindEducationOptionsRequest) ([]dto.EducationOptionResponse, error) {
	if request.RoadmapID <= 0 {
		return nil, fmt.Errorf("roadmap_id must be positive")
	}

	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return nil, fmt.Errorf("find roadmap: %w", err)
	}
	options, err := s.educationOptions(ctx, userID, roadmap, request.ExpandGeography)
	if err != nil {
		return nil, err
	}

	result := make([]dto.EducationOptionResponse, 0, len(options))
	for _, option := range options {
		result = append(result, option.response)
	}
	return result, nil
}

func (s *admissionService) DiagnoseEducationOptions(ctx context.Context, userID int64, request dto.FindEducationOptionsRequest) (dto.EducationOptionsDiagnosisResponse, error) {
	roadmap, err := s.roadmap(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.EducationOptionsDiagnosisResponse{}, err
	}
	diagnosis := dto.EducationOptionsDiagnosisResponse{TargetAdmissionYear: roadmap.UserGoal.TargetAdmissionYear}
	profile, err := s.profiles.FindProfileByUserID(ctx, userID)
	if err != nil {
		return diagnosis, fmt.Errorf("find user profile: %w", err)
	}
	subjects, err := s.profiles.ListUserSubjects(ctx, userID)
	if err != nil {
		return diagnosis, fmt.Errorf("list user subjects: %w", err)
	}
	scores := actualExamScores(subjects)
	if len(scores) == 0 {
		diagnosis.Reason = "missing_exam_results"
		return diagnosis, nil
	}

	allCatalogPrograms, err := s.listProgramsForDiagnosis(ctx, roadmap, profile.RegionID, nil, true, false)
	if err != nil {
		return diagnosis, err
	}
	diagnosis.RulesSourceYear = latestRulesSourceYear(allCatalogPrograms)
	if len(allCatalogPrograms) == 0 {
		diagnosis.Reason = "no_catalog_data"
		return diagnosis, nil
	}

	examCompatiblePrograms, err := s.listProgramsForDiagnosis(ctx, roadmap, profile.RegionID, subjectIDs(scores), true, false)
	if err != nil {
		return diagnosis, err
	}
	if len(examCompatiblePrograms) == 0 {
		diagnosis.Reason = "exam_subjects_mismatch"
		return diagnosis, nil
	}

	if !hasProgramsWithMinimumScores(examCompatiblePrograms, scores) {
		diagnosis.Reason = "minimum_scores_not_met"
		return diagnosis, nil
	}
	employerCompatiblePrograms, err := s.listProgramsForDiagnosis(ctx, roadmap, profile.RegionID, subjectIDs(scores), true, true)
	if err != nil {
		return diagnosis, err
	}
	if len(employerCompatiblePrograms) == 0 {
		diagnosis.Reason = "no_employer_opportunity"
		return diagnosis, nil
	}

	regionalOptions, err := s.educationOptions(ctx, userID, roadmap, false)
	if err != nil {
		return diagnosis, err
	}
	if len(regionalOptions) == 0 && !request.ExpandGeography {
		diagnosis.Reason = "region_restriction"
		return diagnosis, nil
	}
	diagnosis.Reason = "no_options"
	return diagnosis, nil
}

func (s *admissionService) GetAdmissionPlanLimits(ctx context.Context, userID int64, request dto.GetAdmissionPlanRequest) (dto.AdmissionPlanLimitsResponse, error) {
	_, err := s.roadmap(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.AdmissionPlanLimitsResponse{}, err
	}
	rule, err := s.education.GetLatestAdmissionCampaignRule(ctx)
	if err != nil {
		return dto.AdmissionPlanLimitsResponse{}, fmt.Errorf("get latest admission campaign rule: %w", err)
	}
	return admissionPlanLimitsResponse(rule), nil
}

func (s *admissionService) SaveAdmissionPlan(ctx context.Context, userID int64, plan dto.GetAdmissionPlanRequest, request dto.SaveAdmissionPlanRequest) ([]dto.AdmissionApplicationResponse, error) {
	roadmap, err := s.roadmap(ctx, userID, plan.RoadmapID)
	if err != nil {
		return nil, err
	}
	if roadmap.Status != models.RoadmapStatusActive {
		return nil, fmt.Errorf("only an active roadmap can be updated")
	}

	rule, err := s.education.GetLatestAdmissionCampaignRule(ctx)
	if err != nil {
		return nil, fmt.Errorf("get latest admission campaign rule: %w", err)
	}
	profile, err := s.profiles.FindProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user profile: %w", err)
	}
	availableOptions, err := s.educationOptions(ctx, userID, roadmap, profile.WillingToRelocate)
	if err != nil {
		return nil, err
	}
	availableByID := make(map[int64]educationOption, len(availableOptions))
	for _, option := range availableOptions {
		availableByID[option.program.ID] = option
	}

	applications, err := admissionApplications(request.Applications, availableByID, rule)
	if err != nil {
		return nil, err
	}
	savedApplications, err := s.roadmaps.ReplaceAdmissionApplications(ctx, roadmap.ID, applications)
	if err != nil {
		return nil, fmt.Errorf("replace admission applications: %w", err)
	}
	return admissionApplicationResponses(savedApplications), nil
}

func (s *admissionService) GetAdmissionPlan(ctx context.Context, userID int64, request dto.GetAdmissionPlanRequest) ([]dto.AdmissionApplicationResponse, error) {
	roadmap, err := s.roadmap(ctx, userID, request.RoadmapID)
	if err != nil {
		return nil, err
	}
	applications, err := s.roadmaps.ListAdmissionApplications(ctx, roadmap.ID)
	if err != nil {
		return nil, fmt.Errorf("list admission applications: %w", err)
	}
	return admissionApplicationResponses(applications), nil
}

func (s *admissionService) SaveEnrollmentChoice(ctx context.Context, userID int64, plan dto.GetAdmissionPlanRequest, request dto.SaveEnrollmentChoiceRequest) (dto.EnrollmentChoiceResponse, error) {
	roadmap, err := s.roadmap(ctx, userID, plan.RoadmapID)
	if err != nil {
		return dto.EnrollmentChoiceResponse{}, err
	}
	if roadmap.Status != models.RoadmapStatusActive {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("only an active roadmap can be updated")
	}
	if request.Status != models.EnrollmentStatusChosen && request.Status != models.EnrollmentStatusNotEnrolled {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("unsupported enrollment status: %q", request.Status)
	}
	if request.Status == models.EnrollmentStatusChosen && request.AdmissionApplicationID == nil {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("admission_application_id is required for chosen status")
	}
	if request.Status == models.EnrollmentStatusNotEnrolled && request.AdmissionApplicationID != nil {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("admission_application_id must be empty for not_enrolled status")
	}

	applications, err := s.roadmaps.ListAdmissionApplications(ctx, roadmap.ID)
	if err != nil {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("list admission applications: %w", err)
	}
	application := findAdmissionApplication(applications, request.AdmissionApplicationID)
	if request.Status == models.EnrollmentStatusChosen && application == nil {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("admission application does not belong to roadmap")
	}

	choice, err := s.roadmaps.SaveEnrollmentChoice(ctx, models.RoadmapEnrollmentChoice{
		RoadmapID:              roadmap.ID,
		AdmissionApplicationID: request.AdmissionApplicationID,
		Status:                 request.Status,
		EnrollmentYear:         request.EnrollmentYear,
		DecidedAt:              time.Now().UTC(),
	})
	if err != nil {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("save enrollment choice: %w", err)
	}
	if application != nil {
		choice.AdmissionApplication = application
	}

	if request.Status == models.EnrollmentStatusChosen {
		if err := s.attachEmployerOpportunity(ctx, roadmap, *application); err != nil {
			return dto.EnrollmentChoiceResponse{}, err
		}
	}
	if request.Status == models.EnrollmentStatusNotEnrolled {
		if _, err := s.roadmaps.ArchiveRoadmap(ctx, roadmap.ID); err != nil {
			return dto.EnrollmentChoiceResponse{}, fmt.Errorf("archive unsuccessful roadmap: %w", err)
		}
	}
	return enrollmentChoiceResponse(choice), nil
}

type educationOption struct {
	program  models.EducationProgram
	response dto.EducationOptionResponse
}

func (s *admissionService) listProgramsForDiagnosis(ctx context.Context, roadmap models.Roadmap, regionID int64, examSubjectIDs []int64, expandGeography, requireEmployerOpportunity bool) ([]models.EducationProgram, error) {
	programs, err := s.education.ListEducationOptions(ctx, ports.EducationOptionsFilter{
		CareerDirectionID:          roadmap.UserGoal.CareerDirectionID,
		CompanyID:                  roadmap.UserGoal.CareerDirection.CompanyID,
		ExamSubjectIDs:             examSubjectIDs,
		RegionID:                   regionID,
		ExpandGeography:            expandGeography,
		RequireEmployerOpportunity: requireEmployerOpportunity,
	})
	if err != nil {
		return nil, fmt.Errorf("list education options: %w", err)
	}
	return programs, nil
}

func (s *admissionService) educationOptions(ctx context.Context, userID int64, roadmap models.Roadmap, expandGeography bool) ([]educationOption, error) {
	profile, err := s.profiles.FindProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user profile: %w", err)
	}
	subjects, err := s.profiles.ListUserSubjects(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user subjects: %w", err)
	}
	scores := actualExamScores(subjects)
	if len(scores) == 0 {
		return nil, fmt.Errorf("at least one passed exam with an actual score is required")
	}

	programs, err := s.listProgramsForDiagnosis(ctx, roadmap, profile.RegionID, subjectIDs(scores), expandGeography, true)
	if err != nil {
		return nil, err
	}

	result := make([]educationOption, 0, len(programs))
	for _, program := range programs {
		combination, totalScore, ok := matchingExamCombination(program.ExamCombinations, scores)
		if !ok {
			continue
		}
		budgetScore, paidScore, passingScoreSourceYear := latestAdmissionScores(program.AdmissionScores)
		minimumTotalScore := minimumTotalScore(combination)
		result = append(result, educationOption{
			program: program,
			response: dto.EducationOptionResponse{
				UniversityID:           program.UniversityID,
				UniversityName:         program.University.Name,
				UniversityRegion:       program.University.Region.Name,
				EducationProgramID:     program.ID,
				ProgramCode:            program.Code,
				ProgramName:            program.Name,
				AdmissionYear:          combination.AdmissionYear,
				RulesSourceYear:        combination.AdmissionYear,
				RequiredSubjects:       combinationSubjectNames(combination),
				MinimumTotalScore:      minimumTotalScore,
				BudgetPassingScore:     budgetScore,
				PaidPassingScore:       paidScore,
				PassingScoreSourceYear: passingScoreSourceYear,
				Explanation:            optionExplanation(totalScore, budgetScore, paidScore),
			},
		})
	}
	return result, nil
}

func (s *admissionService) roadmap(ctx context.Context, userID, roadmapID int64) (models.Roadmap, error) {
	if roadmapID <= 0 {
		return models.Roadmap{}, fmt.Errorf("roadmap_id must be positive")
	}
	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, roadmapID)
	if err != nil {
		return models.Roadmap{}, fmt.Errorf("find roadmap: %w", err)
	}
	return roadmap, nil
}

func actualExamScores(subjects []models.UserSubject) map[int64]int16 {
	scores := make(map[int64]int16)
	for _, subject := range subjects {
		if subject.Status == models.SubjectStatusPassed && subject.ActualScore != nil {
			scores[subject.ExamSubjectID] = *subject.ActualScore
		}
	}
	return scores
}

func subjectIDs(scores map[int64]int16) []int64 {
	ids := make([]int64, 0, len(scores))
	for subjectID := range scores {
		ids = append(ids, subjectID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func matchingExamCombination(combinations []models.ExamCombination, scores map[int64]int16) (models.ExamCombination, int16, bool) {
	var latestYear int16
	for _, combination := range combinations {
		if combination.AdmissionYear > latestYear {
			latestYear = combination.AdmissionYear
		}
	}
	if latestYear == 0 {
		return models.ExamCombination{}, 0, false
	}
	sort.SliceStable(combinations, func(i, j int) bool { return combinations[i].ID < combinations[j].ID })
	for _, combination := range combinations {
		if combination.AdmissionYear != latestYear {
			continue
		}
		var totalScore int16
		matches := len(combination.Items) > 0
		for _, item := range combination.Items {
			score, exists := scores[item.ExamSubjectID]
			if !exists || (item.MinScore != nil && score < *item.MinScore) {
				matches = false
				break
			}
			totalScore += score
		}
		if matches {
			return combination, totalScore, true
		}
	}
	return models.ExamCombination{}, 0, false
}

func hasProgramsWithMinimumScores(programs []models.EducationProgram, scores map[int64]int16) bool {
	for _, program := range programs {
		if _, _, ok := matchingExamCombination(program.ExamCombinations, scores); ok {
			return true
		}
	}
	return false
}

func latestRulesSourceYear(programs []models.EducationProgram) *int16 {
	var latest int16
	for _, program := range programs {
		for _, combination := range program.ExamCombinations {
			if combination.AdmissionYear > latest {
				latest = combination.AdmissionYear
			}
		}
	}
	if latest == 0 {
		return nil
	}
	return &latest
}

func combinationSubjectNames(combination models.ExamCombination) []string {
	names := make([]string, 0, len(combination.Items))
	for _, item := range combination.Items {
		names = append(names, item.ExamSubject.Name)
	}
	sort.Strings(names)
	return names
}

func minimumTotalScore(combination models.ExamCombination) *int16 {
	var total int16
	for _, item := range combination.Items {
		if item.MinScore == nil {
			return nil
		}
		total += *item.MinScore
	}
	return &total
}

func latestAdmissionScores(scores []models.AdmissionScoreHistory) (*int16, *int16, *int16) {
	var latest *models.AdmissionScoreHistory
	for _, score := range scores {
		if latest == nil || score.AdmissionYear > latest.AdmissionYear {
			candidate := score
			latest = &candidate
		}
	}
	if latest == nil {
		return nil, nil, nil
	}
	year := latest.AdmissionYear
	return latest.BudgetPassingScore, latest.PaidPassingScore, &year
}

func optionExplanation(totalScore int16, budgetScore, paidScore *int16) []string {
	explanation := []string{"Сданы необходимые ЕГЭ", "Минимальные баллы по предметам соблюдены"}
	if budgetScore != nil {
		if totalScore >= *budgetScore {
			explanation = append(explanation, "Суммарный балл соответствует бюджетному проходному баллу")
		} else {
			explanation = append(explanation, "Суммарный балл ниже бюджетного проходного балла")
		}
	}
	if paidScore != nil {
		if totalScore >= *paidScore {
			explanation = append(explanation, "Суммарный балл соответствует платному проходному баллу")
		} else {
			explanation = append(explanation, "Суммарный балл ниже платного проходного балла")
		}
	}
	return explanation
}

func admissionPlanLimitsResponse(rule models.AdmissionCampaignRule) dto.AdmissionPlanLimitsResponse {
	return dto.AdmissionPlanLimitsResponse{
		AdmissionYear:            rule.AdmissionYear,
		RulesSourceYear:          rule.AdmissionYear,
		MaxUniversities:          rule.MaxUniversities,
		MaxProgramsPerUniversity: rule.MaxProgramsPerUniversity,
	}
}

func admissionApplications(inputs []dto.AdmissionPlanItemInput, available map[int64]educationOption, rule models.AdmissionCampaignRule) ([]models.RoadmapAdmissionApplication, error) {
	programsByUniversity := make(map[int64]int)
	seenPrograms := make(map[int64]struct{}, len(inputs))
	applications := make([]models.RoadmapAdmissionApplication, 0, len(inputs))
	for _, input := range inputs {
		if input.EducationProgramID <= 0 {
			return nil, fmt.Errorf("education_program_id must be positive")
		}
		if _, exists := seenPrograms[input.EducationProgramID]; exists {
			return nil, fmt.Errorf("education program %d is repeated", input.EducationProgramID)
		}
		option, exists := available[input.EducationProgramID]
		if !exists {
			return nil, fmt.Errorf("education program %d is unavailable for current exam results", input.EducationProgramID)
		}
		seenPrograms[input.EducationProgramID] = struct{}{}
		programsByUniversity[option.program.UniversityID]++
		if programsByUniversity[option.program.UniversityID] > int(rule.MaxProgramsPerUniversity) {
			return nil, fmt.Errorf("admission plan exceeds program limit for university %d", option.program.UniversityID)
		}
		applications = append(applications, models.RoadmapAdmissionApplication{
			EducationProgramID: input.EducationProgramID,
			Status:             models.AdmissionApplicationStatusPlanned,
			CreatedAt:          time.Now().UTC(),
		})
	}
	if len(programsByUniversity) > int(rule.MaxUniversities) {
		return nil, fmt.Errorf("admission plan exceeds university limit")
	}
	return applications, nil
}

func admissionApplicationResponses(applications []models.RoadmapAdmissionApplication) []dto.AdmissionApplicationResponse {
	result := make([]dto.AdmissionApplicationResponse, 0, len(applications))
	for _, application := range applications {
		result = append(result, dto.AdmissionApplicationResponse{
			ID:                 application.ID,
			UniversityID:       application.EducationProgram.UniversityID,
			UniversityName:     application.EducationProgram.University.Name,
			EducationProgramID: application.EducationProgramID,
			ProgramName:        application.EducationProgram.Name,
			Status:             application.Status,
		})
	}
	return result
}

func findAdmissionApplication(applications []models.RoadmapAdmissionApplication, applicationID *int64) *models.RoadmapAdmissionApplication {
	if applicationID == nil {
		return nil
	}
	for index := range applications {
		if applications[index].ID == *applicationID {
			return &applications[index]
		}
	}
	return nil
}

func (s *admissionService) attachEmployerOpportunity(ctx context.Context, roadmap models.Roadmap, application models.RoadmapAdmissionApplication) error {
	direction := roadmap.UserGoal.CareerDirection
	opportunity, err := s.careers.FindActiveOpportunity(ctx, direction.CompanyID, direction.ID, application.EducationProgram.University.RegionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find employer opportunity: %w", err)
	}

	steps := make([]models.RoadmapStep, 0)
	opportunityAttached := false
	for _, step := range roadmap.Steps {
		if step.Status != models.RoadmapStepStatusPending && step.Status != models.RoadmapStepStatusActive {
			continue
		}
		step.ID = 0
		step.RoadmapID = 0
		if step.StepType == models.RoadmapStepTypeEmployerExperience {
			opportunityID := opportunity.ID
			step.CompanyOpportunityID = &opportunityID
			step.Title = opportunity.Name
			step.Description = opportunity.Description
			step.ActionURL = opportunity.URL
			step.TargetStudyYear = &opportunity.MinStudyYear
			opportunityAttached = true
		}
		steps = append(steps, step)
	}
	if !opportunityAttached {
		return nil
	}
	if _, err := s.roadmaps.ReplaceUncompletedSteps(ctx, roadmap.ID, steps); err != nil {
		return fmt.Errorf("refine uncompleted roadmap steps: %w", err)
	}
	return nil
}

func enrollmentChoiceResponse(choice models.RoadmapEnrollmentChoice) dto.EnrollmentChoiceResponse {
	response := dto.EnrollmentChoiceResponse{
		Status:                 choice.Status,
		AdmissionApplicationID: choice.AdmissionApplicationID,
		EnrollmentYear:         choice.EnrollmentYear,
	}
	if choice.AdmissionApplication != nil {
		response.UniversityName = choice.AdmissionApplication.EducationProgram.University.Name
		response.EducationProgramName = choice.AdmissionApplication.EducationProgram.Name
	}
	return response
}
