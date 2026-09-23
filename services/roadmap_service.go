package services

import (
	"context"

	"efr_bot/dto"
)

// RoadmapService создаёт и постепенно уточняет персональный roadmap пользователя.
type RoadmapService interface {
	// CreateRoadmap выбирает шаблон для подтверждённой цели и создаёт снимок общих шагов.
	// Вход: dto.CreateRoadmapRequest и идентификатор пользователя. Выход: dto.RoadmapResponse.
	CreateRoadmap(ctx context.Context, userID int64, request dto.CreateRoadmapRequest) (dto.RoadmapResponse, error)

	// GetRoadmap возвращает roadmap вместе с текущим следующим действием.
	// Вход: dto.GetRoadmapRequest и идентификатор пользователя. Выход: dto.RoadmapResponse.
	GetRoadmap(ctx context.Context, userID int64, request dto.GetRoadmapRequest) (dto.RoadmapResponse, error)

	// UpdateRoadmapStep обновляет статус одного шага.
	// Вход: dto.GetRoadmapStepRequest и dto.UpdateRoadmapStepRequest. Выход: dto.RoadmapStepResponse.
	UpdateRoadmapStep(ctx context.Context, userID int64, step dto.GetRoadmapStepRequest, request dto.UpdateRoadmapStepRequest) (dto.RoadmapStepResponse, error)

	// ReconsiderTrajectory архивирует текущий roadmap и запускает сценарий пересмотра траектории.
	// Вход: dto.GetRoadmapRequest и dto.ReconsiderTrajectoryRequest. Выход: dto.RoadmapResponse новой версии.
	ReconsiderTrajectory(ctx context.Context, userID int64, roadmap dto.GetRoadmapRequest, request dto.ReconsiderTrajectoryRequest) (dto.RoadmapResponse, error)

	// GetEmployerOpportunity возвращает назначенную возможность работодателя после выбора вуза.
	// Вход: dto.GetRoadmapRequest и идентификатор пользователя. Выход: dto.CompanyOpportunityResponse.
	GetEmployerOpportunity(ctx context.Context, userID int64, request dto.GetRoadmapRequest) (dto.CompanyOpportunityResponse, error)
}
