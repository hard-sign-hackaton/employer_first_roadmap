package repositories

import (
	"context"

	"efr_bot/models"
)

// RoadmapRepository хранит цели, шаблоны, снимки roadmap и решения приёмной кампании.
type RoadmapRepository interface {
	// CreateGoal сохраняет подтверждённую цель пользователя.
	// Вход: models.UserGoal. Выход: сохранённый models.UserGoal.
	CreateGoal(ctx context.Context, goal models.UserGoal) (models.UserGoal, error)

	// FindGoalByID получает цель, принадлежащую конкретному пользователю.
	// Вход: идентификаторы пользователя и цели. Выход: models.UserGoal с направлением и компанией.
	FindGoalByID(ctx context.Context, userID, goalID int64) (models.UserGoal, error)

	// FindActiveRoadmapByUserID возвращает текущий активный roadmap пользователя.
	// Вход: идентификатор пользователя. Выход: models.Roadmap с шагами и целью.
	FindActiveRoadmapByUserID(ctx context.Context, userID int64) (models.Roadmap, error)

	// FindRoadmapByID получает roadmap, принадлежащий конкретному пользователю.
	// Вход: идентификаторы пользователя и roadmap. Выход: models.Roadmap с шагами и целью.
	FindRoadmapByID(ctx context.Context, userID, roadmapID int64) (models.Roadmap, error)

	// FindActiveTemplateForDirection выбирает актуальный шаблон roadmap для направления.
	// Вход: идентификатор карьерного направления. Выход: models.RoadmapTemplate со шагами.
	FindActiveTemplateForDirection(ctx context.Context, careerDirectionID int64) (models.RoadmapTemplate, error)

	// CreateRoadmapWithSteps создаёт снимок roadmap и его персональные шаги как одну операцию.
	// Вход: models.Roadmap и список models.RoadmapStep. Выход: сохранённый models.Roadmap.
	CreateRoadmapWithSteps(ctx context.Context, roadmap models.Roadmap, steps []models.RoadmapStep) (models.Roadmap, error)

	// UpdateRoadmapStepStatus обновляет статус и время завершения одного шага.
	// Вход: идентификаторы roadmap и шага, новый статус. Выход: обновлённый models.RoadmapStep.
	UpdateRoadmapStepStatus(ctx context.Context, roadmapID, stepID int64, status string) (models.RoadmapStep, error)

	// ReplaceUncompletedSteps заменяет только будущие незавершённые шаги при уточнении roadmap.
	// Вход: идентификатор roadmap и новый список шагов. Выход: сохранённый список models.RoadmapStep.
	ReplaceUncompletedSteps(ctx context.Context, roadmapID int64, steps []models.RoadmapStep) ([]models.RoadmapStep, error)

	// ReplaceAdmissionApplications сохраняет план подачи документов как одну согласованную операцию.
	// Вход: идентификатор roadmap и список models.RoadmapAdmissionApplication. Выход: сохранённый список.
	ReplaceAdmissionApplications(ctx context.Context, roadmapID int64, applications []models.RoadmapAdmissionApplication) ([]models.RoadmapAdmissionApplication, error)

	// ListAdmissionApplications возвращает сохранённый план подачи с ОП и вузами.
	// Вход: идентификатор roadmap. Выход: список models.RoadmapAdmissionApplication.
	ListAdmissionApplications(ctx context.Context, roadmapID int64) ([]models.RoadmapAdmissionApplication, error)

	// SaveEnrollmentChoice сохраняет единственное итоговое решение после приёмной кампании.
	// Вход: models.RoadmapEnrollmentChoice. Выход: сохранённая models.RoadmapEnrollmentChoice.
	SaveEnrollmentChoice(ctx context.Context, choice models.RoadmapEnrollmentChoice) (models.RoadmapEnrollmentChoice, error)

	// ArchiveRoadmap архивирует текущую версию roadmap перед пересмотром траектории.
	// Вход: идентификатор roadmap. Выход: архивированный models.Roadmap.
	ArchiveRoadmap(ctx context.Context, roadmapID int64) (models.Roadmap, error)
}
