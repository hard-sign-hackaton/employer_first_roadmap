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

func (s *roadmapService) RestartActiveRoadmap(ctx context.Context, userID int64) error {
	if err := s.roadmaps.ResetUserData(ctx, userID); err != nil {
		return fmt.Errorf("reset user data: %w", err)
	}
	return nil
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

func (s *roadmapService) ArchiveRoadmapForRevision(ctx context.Context, userID int64, request dto.GetRoadmapRequest) error {
	if request.RoadmapID <= 0 {
		return fmt.Errorf("roadmap_id must be positive")
	}
	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return fmt.Errorf("find roadmap for revision: %w", err)
	}
	if roadmap.Status != models.RoadmapStatusActive {
		return fmt.Errorf("only an active roadmap can be revised")
	}
	if _, err := s.roadmaps.ArchiveRoadmap(ctx, roadmap.ID); err != nil {
		return fmt.Errorf("archive roadmap for revision: %w", err)
	}
	return nil
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
	return dto.CompanyOpportunityResponse{
		ID:                opportunity.ID,
		CompanyName:       opportunity.Company.Name,
		Type:              opportunity.Type,
		Name:              opportunity.Name,
		Description:       opportunity.Description,
		URL:               opportunity.URL,
		CompanyWebsiteURL: opportunity.Company.WebsiteURL,
		MinStudyYear:      opportunity.MinStudyYear,
		IsAvailable:       opportunity.IsActive,
	}, nil
}

func (s *roadmapService) SubmitEmployerApplication(ctx context.Context, userID int64, request dto.GetRoadmapRequest) (dto.EmployerApplicationResponse, error) {
	roadmap, err := s.roadmaps.FindRoadmapByID(ctx, userID, request.RoadmapID)
	if err != nil {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("find roadmap: %w", err)
	}
	if roadmap.Status != models.RoadmapStatusActive {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("only an active roadmap can submit an employer application")
	}
	if roadmap.EnrollmentChoice == nil || roadmap.EnrollmentChoice.Status != models.EnrollmentStatusChosen {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("an enrolled education program is required before applying to employer")
	}
	var opportunityID int64
	for _, step := range roadmap.Steps {
		if step.StepType == models.RoadmapStepTypeEmployerExperience && step.CompanyOpportunityID != nil {
			opportunityID = *step.CompanyOpportunityID
		}
	}
	if opportunityID == 0 {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("an employer opportunity is not selected yet")
	}
	saved, err := s.roadmaps.SaveEmployerApplication(ctx, models.RoadmapEmployerApplication{RoadmapID: roadmap.ID, CompanyOpportunityID: opportunityID, Status: "submitted", SubmittedAt: time.Now().UTC()})
	if err != nil {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("save employer application: %w", err)
	}
	return dto.EmployerApplicationResponse{RoadmapID: saved.RoadmapID, CompanyOpportunityID: saved.CompanyOpportunityID, Status: saved.Status}, nil
}

func (s *roadmapService) GetEmployerFeedback(ctx context.Context, userID int64) (dto.EmployerApplicationResponse, error) {
	application, err := s.roadmaps.FindLatestEmployerApplicationByUserID(ctx, userID)
	if err != nil {
		return dto.EmployerApplicationResponse{}, fmt.Errorf("find employer application: %w", err)
	}
	return employerApplicationResponse(application), nil
}

func (s *roadmapService) ListEmployerApplications(ctx context.Context, userID int64) ([]dto.EmployerApplicationResponse, error) {
	applications, err := s.roadmaps.ListEmployerApplicationsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list employer applications: %w", err)
	}
	result := make([]dto.EmployerApplicationResponse, 0, len(applications))
	for _, application := range applications {
		result = append(result, employerApplicationResponse(application))
	}
	return result, nil
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
		response := employerApplicationResponse(*roadmap.EmployerApplication)
		employerApplication = &response
	}
	return dto.RoadmapResponse{
		ID:     roadmap.ID,
		GoalID: roadmap.UserGoalID,
		Goal: dto.RoadmapGoalResponse{
			CompanyName:         roadmap.UserGoal.CareerDirection.Company.Name,
			CareerDirectionName: roadmap.UserGoal.CareerDirection.Name,
			TargetAdmissionYear: roadmap.UserGoal.TargetAdmissionYear,
		},
		Status:              roadmap.Status,
		Steps:               steps,
		NextAction:          nextAction,
		EnrollmentChoice:    enrollmentChoice,
		EmployerApplication: employerApplication,
	}
}

func employerApplicationResponse(application models.RoadmapEmployerApplication) dto.EmployerApplicationResponse {
	response := dto.EmployerApplicationResponse{
		RoadmapID:            application.RoadmapID,
		CompanyOpportunityID: application.CompanyOpportunityID,
		CompanyName:          application.CompanyOpportunity.Company.Name,
		OpportunityName:      application.CompanyOpportunity.Name,
		Status:               application.Status,
		History:              make([]dto.EmployerFeedbackResponse, 0, len(application.Feedbacks)),
	}
	for _, feedback := range application.Feedbacks {
		response.History = append(response.History, dto.EmployerFeedbackResponse{Status: feedback.Status, Message: feedback.Message, Contact: feedback.Contact, CreatedAt: feedback.CreatedAt})
		response.Status = feedback.Status
		response.Message = feedback.Message
		response.Contact = feedback.Contact
		updatedAt := feedback.CreatedAt
		response.UpdatedAt = &updatedAt
	}
	return response
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
