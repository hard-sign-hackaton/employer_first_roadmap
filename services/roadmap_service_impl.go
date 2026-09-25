package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/services/ports"
)

// roadmapService создаёт и обновляет персональные roadmap по шаблонам направлений.
type roadmapService struct {
	roadmaps ports.RoadmapStore
	careers  ports.CareerStore
}

var _ RoadmapService = (*roadmapService)(nil)

// NewRoadmapService создаёт сервис персональных roadmap.
func NewRoadmapService(roadmaps ports.RoadmapStore, careers ports.CareerStore) RoadmapService {
	return &roadmapService{
		roadmaps: roadmaps,
		careers:  careers,
	}
}

func (s *roadmapService) CreateRoadmap(ctx context.Context, userID int64, request dto.CreateRoadmapRequest) (dto.RoadmapResponse, error) {
	if request.GoalID <= 0 {
		return dto.RoadmapResponse{}, fmt.Errorf("goal_id must be positive")
	}

	goal, err := s.roadmaps.FindGoalByID(ctx, userID, request.GoalID)
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("find user goal: %w", err)
	}
	template, err := s.roadmaps.FindActiveTemplateForDirection(ctx, goal.CareerDirectionID)
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("find active roadmap template: %w", err)
	}

	roadmap, err := s.roadmaps.CreateRoadmapWithSteps(ctx, models.Roadmap{
		UserGoalID:        goal.ID,
		RoadmapTemplateID: template.ID,
		Status:            models.RoadmapStatusActive,
		CreatedAt:         time.Now().UTC(),
	}, roadmapSteps(template))
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("create roadmap: %w", err)
	}
	return roadmapResponse(roadmap), nil
}

func (s *roadmapService) GetRoadmap(ctx context.Context, userID int64, request dto.GetRoadmapRequest) (dto.RoadmapResponse, error) {
	if request.RoadmapID <= 0 {
		return dto.RoadmapResponse{}, fmt.Errorf("roadmap_id must be positive")
	}

	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("find roadmap: %w", err)
	}
	return roadmapResponse(roadmap), nil
}

func (s *roadmapService) GetActiveRoadmap(ctx context.Context, userID int64) (dto.RoadmapResponse, error) {
	roadmap, err := s.roadmaps.FindActiveRoadmapByUserID(ctx, userID)
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("find active roadmap: %w", err)
	}
	return roadmapResponse(roadmap), nil
}

func (s *roadmapService) UpdateRoadmapStep(ctx context.Context, userID int64, step dto.GetRoadmapStepRequest, request dto.UpdateRoadmapStepRequest) (dto.RoadmapStepResponse, error) {
	if step.RoadmapID <= 0 || step.StepID <= 0 {
		return dto.RoadmapStepResponse{}, fmt.Errorf("roadmap_id and step_id must be positive")
	}
	if !isRoadmapStepStatus(request.Status) {
		return dto.RoadmapStepResponse{}, fmt.Errorf("unsupported roadmap step status: %q", request.Status)
	}
	if _, err := s.roadmaps.FindRoadmapByID(ctx, userID, step.RoadmapID); err != nil {
		return dto.RoadmapStepResponse{}, fmt.Errorf("find roadmap: %w", err)
	}

	var updatedStep models.RoadmapStep
	var err error
	if request.Status == models.RoadmapStepStatusCompleted {
		updatedStep, err = s.roadmaps.CompleteRoadmapStep(ctx, step.RoadmapID, step.StepID)
	} else {
		updatedStep, err = s.roadmaps.UpdateRoadmapStepStatus(ctx, step.RoadmapID, step.StepID, request.Status)
	}
	if err != nil {
		return dto.RoadmapStepResponse{}, fmt.Errorf("update roadmap step: %w", err)
	}
	return roadmapStepResponse(updatedStep), nil
}

func (s *roadmapService) ReconsiderTrajectory(ctx context.Context, userID int64, request dto.GetRoadmapRequest, reconsider dto.ReconsiderTrajectoryRequest) (dto.RoadmapResponse, error) {
	if request.RoadmapID <= 0 {
		return dto.RoadmapResponse{}, fmt.Errorf("roadmap_id must be positive")
	}
	if strings.TrimSpace(reconsider.Reason) == "" {
		return dto.RoadmapResponse{}, fmt.Errorf("reason must not be empty")
	}

	currentRoadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("find roadmap for reconsideration: %w", err)
	}
	if currentRoadmap.Status != models.RoadmapStatusActive {
		return dto.RoadmapResponse{}, fmt.Errorf("only an active roadmap can be reconsidered")
	}

	template, err := s.roadmaps.FindActiveTemplateForDirection(ctx, currentRoadmap.UserGoal.CareerDirectionID)
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("find active roadmap template: %w", err)
	}
	if _, err := s.roadmaps.ArchiveRoadmap(ctx, currentRoadmap.ID); err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("archive current roadmap: %w", err)
	}

	newRoadmap, err := s.roadmaps.CreateRoadmapWithSteps(ctx, models.Roadmap{
		UserGoalID:        currentRoadmap.UserGoalID,
		RoadmapTemplateID: template.ID,
		Status:            models.RoadmapStatusActive,
		CreatedAt:         time.Now().UTC(),
	}, roadmapSteps(template))
	if err != nil {
		return dto.RoadmapResponse{}, fmt.Errorf("create reconsidered roadmap: %w", err)
	}
	return roadmapResponse(newRoadmap), nil
}

func (s *roadmapService) GetEmployerOpportunity(ctx context.Context, userID int64, request dto.GetRoadmapRequest) (dto.CompanyOpportunityResponse, error) {
	if request.RoadmapID <= 0 {
		return dto.CompanyOpportunityResponse{}, fmt.Errorf("roadmap_id must be positive")
	}

	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.CompanyOpportunityResponse{}, fmt.Errorf("find roadmap: %w", err)
	}
	if roadmap.EnrollmentChoice == nil || roadmap.EnrollmentChoice.Status != models.EnrollmentStatusChosen || roadmap.EnrollmentChoice.AdmissionApplication == nil {
		return dto.CompanyOpportunityResponse{}, fmt.Errorf("an enrolled education program is required before selecting an employer opportunity")
	}

	var opportunity *models.CompanyOpportunity
	for index := range roadmap.Steps {
		step := &roadmap.Steps[index]
		if step.StepType == models.RoadmapStepTypeEmployerExperience && step.CompanyOpportunity != nil {
			opportunity = step.CompanyOpportunity
			break
		}
	}
	if opportunity == nil {
		return dto.CompanyOpportunityResponse{}, fmt.Errorf("no employer opportunity is configured for the selected program")
	}
	if roadmap.EnrollmentChoice.CurrentStudyYear < opportunity.MinStudyYear {
		return dto.CompanyOpportunityResponse{}, fmt.Errorf("employer opportunity is available from study year %d", opportunity.MinStudyYear)
	}
	return dto.CompanyOpportunityResponse{
		ID:           opportunity.ID,
		Type:         opportunity.Type,
		Name:         opportunity.Name,
		Description:  opportunity.Description,
		URL:          opportunity.URL,
		MinStudyYear: opportunity.MinStudyYear,
		IsAvailable:  opportunity.IsActive,
	}, nil
}

func (s *roadmapService) AdvanceStudyYear(ctx context.Context, userID int64, request dto.GetRoadmapRequest) (dto.EnrollmentChoiceResponse, error) {
	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("find roadmap: %w", err)
	}
	choice := roadmap.EnrollmentChoice
	if roadmap.Status != models.RoadmapStatusActive || choice == nil || choice.Status != models.EnrollmentStatusChosen {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("an active enrolled roadmap is required")
	}
	if choice.CurrentStudyYear >= 6 {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("maximum study year has already been reached")
	}
	choice.CurrentStudyYear++
	choice.DecidedAt = time.Now().UTC()
	saved, err := s.roadmaps.SaveEnrollmentChoice(ctx, *choice)
	if err != nil {
		return dto.EnrollmentChoiceResponse{}, fmt.Errorf("update study year: %w", err)
	}
	return enrollmentChoiceResponse(saved), nil
}

func (s *roadmapService) SubmitEmployerApplication(ctx context.Context, userID int64, request dto.GetRoadmapRequest) (dto.EmployerApplicationResponse, error) {
	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("find roadmap: %w", err)
	}
	if roadmap.Status != models.RoadmapStatusActive {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("only an active roadmap can submit an employer application")
	}
	var opportunityID int64
	for _, step := range roadmap.Steps {
		if step.StepType == models.RoadmapStepTypeEmployerExperience && step.Status == models.RoadmapStepStatusCompleted && step.CompanyOpportunityID != nil {
			opportunityID = *step.CompanyOpportunityID
		}
	}
	if opportunityID == 0 {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("complete an employer opportunity before applying")
	}
	var nextStep *models.RoadmapStep
	for index := range roadmap.Steps {
		step := &roadmap.Steps[index]
		if step.Status == models.RoadmapStepStatusActive || step.Status == models.RoadmapStepStatusPending {
			nextStep = step
			break
		}
	}
	if nextStep == nil || nextStep.StepType != models.RoadmapStepTypeApplyToEmployer {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("employer application is not the current roadmap step")
	}
	saved, err := s.roadmaps.SaveEmployerApplication(ctx, models.RoadmapEmployerApplication{RoadmapID: roadmap.ID, CompanyOpportunityID: opportunityID, Status: "submitted", SubmittedAt: time.Now().UTC()})
	if err != nil {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("save employer application: %w", err)
	}
	return dto.EmployerApplicationResponse{CompanyOpportunityID: saved.CompanyOpportunityID, Status: saved.Status}, nil
}

func roadmapSteps(template models.RoadmapTemplate) []models.RoadmapStep {
	steps := make([]models.RoadmapStep, 0, len(template.Steps))
	for index, templateStep := range template.Steps {
		templateStepID := templateStep.ID
		status := models.RoadmapStepStatusPending
		if index == 0 {
			status = models.RoadmapStepStatusActive
		}
		steps = append(steps, models.RoadmapStep{
			TemplateStepID: &templateStepID,
			OrderNo:        templateStep.OrderNo,
			StepType:       templateStep.StepType,
			Title:          templateStep.Title,
			Description:    templateStep.Description,
			Status:         status,
		})
	}
	return steps
}

func roadmapResponse(roadmap models.Roadmap) dto.RoadmapResponse {
	steps := make([]dto.RoadmapStepResponse, 0, len(roadmap.Steps))
	var nextAction *dto.RoadmapStepResponse
	var enrollmentChoice *dto.EnrollmentChoiceResponse
	var employerApplication *dto.EmployerApplicationResponse
	for _, step := range roadmap.Steps {
		response := roadmapStepResponse(step)
		steps = append(steps, response)
		if nextAction == nil && (step.Status == models.RoadmapStepStatusActive || step.Status == models.RoadmapStepStatusPending) {
			next := response
			nextAction = &next
		}
	}
	if roadmap.EnrollmentChoice != nil {
		response := enrollmentChoiceResponse(*roadmap.EnrollmentChoice)
		enrollmentChoice = &response
	}
	if roadmap.EmployerApplication != nil {
		response := dto.EmployerApplicationResponse{CompanyOpportunityID: roadmap.EmployerApplication.CompanyOpportunityID, Status: roadmap.EmployerApplication.Status}
		employerApplication = &response
	}
	return dto.RoadmapResponse{
		ID:                  roadmap.ID,
		GoalID:              roadmap.UserGoalID,
		Status:              roadmap.Status,
		Steps:               steps,
		NextAction:          nextAction,
		EnrollmentChoice:    enrollmentChoice,
		EmployerApplication: employerApplication,
	}
}

func roadmapStepResponse(step models.RoadmapStep) dto.RoadmapStepResponse {
	return dto.RoadmapStepResponse{
		ID:              step.ID,
		OrderNo:         step.OrderNo,
		StepType:        step.StepType,
		Title:           step.Title,
		Description:     step.Description,
		ActionURL:       step.ActionURL,
		TargetStudyYear: step.TargetStudyYear,
		Status:          step.Status,
		CompletedAt:     step.CompletedAt,
	}
}

func isRoadmapStepStatus(status string) bool {
	switch status {
	case models.RoadmapStepStatusPending,
		models.RoadmapStepStatusActive,
		models.RoadmapStepStatusCompleted,
		models.RoadmapStepStatusSkipped:
		return true
	default:
		return false
	}
}
