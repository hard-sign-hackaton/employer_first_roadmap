package ports

import (
	"context"

	"efr_bot/models"
)

// RoadmapStore предоставляет сервисам цели, шаблоны и персональные roadmap.
type RoadmapStore interface {
	CreateGoal(ctx context.Context, goal models.UserGoal) (models.UserGoal, error)
	FindGoalByID(ctx context.Context, userID, goalID int64) (models.UserGoal, error)
	FindActiveRoadmapByUserID(ctx context.Context, userID int64) (models.Roadmap, error)
	FindRoadmapByID(ctx context.Context, userID, roadmapID int64) (models.Roadmap, error)
	FindActiveTemplateForDirection(ctx context.Context, careerDirectionID int64) (models.RoadmapTemplate, error)
	CreateRoadmapWithSteps(ctx context.Context, roadmap models.Roadmap, steps []models.RoadmapStep) (models.Roadmap, error)
	UpdateRoadmapStepStatus(ctx context.Context, roadmapID, stepID int64, status string) (models.RoadmapStep, error)
	CompleteRoadmapStep(ctx context.Context, roadmapID, stepID int64) (models.RoadmapStep, error)
	ReplaceUncompletedSteps(ctx context.Context, roadmapID int64, steps []models.RoadmapStep) ([]models.RoadmapStep, error)
	ReplaceAdmissionApplications(ctx context.Context, roadmapID int64, applications []models.RoadmapAdmissionApplication) ([]models.RoadmapAdmissionApplication, error)
	ListAdmissionApplications(ctx context.Context, roadmapID int64) ([]models.RoadmapAdmissionApplication, error)
	SaveEnrollmentChoice(ctx context.Context, choice models.RoadmapEnrollmentChoice) (models.RoadmapEnrollmentChoice, error)
	SaveEmployerApplication(ctx context.Context, application models.RoadmapEmployerApplication) (models.RoadmapEmployerApplication, error)
	ArchiveRoadmap(ctx context.Context, roadmapID int64) (models.Roadmap, error)
	ResetUserData(ctx context.Context, userID int64) error
}
