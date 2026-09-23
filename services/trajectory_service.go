package services

import (
	"context"

	"efr_bot/dto"
)

// TrajectoryService отвечает за выбор работодателя, карьерного направления и набора ЕГЭ.
type TrajectoryService interface {
	// FindCompanies ищет работодателей в каталоге минимального опроса.
	// Вход: dto.FindCompaniesRequest. Выход: список dto.CompanyCatalogItemResponse.
	FindCompanies(ctx context.Context, request dto.FindCompaniesRequest) ([]dto.CompanyCatalogItemResponse, error)

	// RecommendCompanies ранжирует работодателей по результатам полного опроса.
	// Вход: dto.RecommendCompaniesRequest и идентификатор пользователя. Выход: список dto.RecommendedCompanyResponse.
	RecommendCompanies(ctx context.Context, userID int64, request dto.RecommendCompaniesRequest) ([]dto.RecommendedCompanyResponse, error)

	// SelectCompany подтверждает выбор работодателя для дальнейшего сценария.
	// Вход: dto.SelectCompanyRequest. Выход: dto.CompanyCatalogItemResponse.
	SelectCompany(ctx context.Context, userID int64, request dto.SelectCompanyRequest) (dto.CompanyCatalogItemResponse, error)

	// GetCareerDirections возвращает направления выбранной компании.
	// Для 11 класса сервис исключает направления без пути с выбранными ЕГЭ.
	// Вход: dto.GetCareerDirectionsRequest и идентификатор пользователя. Выход: список dto.CareerDirectionResponse.
	GetCareerDirections(ctx context.Context, userID int64, request dto.GetCareerDirectionsRequest) ([]dto.CareerDirectionResponse, error)

	// GetRecommendedExamSets возвращает наборы ЕГЭ для направления ученика 9–10 класса
	// или для одиннадцатиклассника, который ещё не выбрал ЕГЭ.
	// Вход: dto.GetRecommendedExamSetsRequest. Выход: список dto.RecommendedExamSetResponse.
	GetRecommendedExamSets(ctx context.Context, request dto.GetRecommendedExamSetsRequest) ([]dto.RecommendedExamSetResponse, error)

	// ConfirmGoal создаёт цель «компания + направление + набор ЕГЭ».
	// Вход: dto.ConfirmGoalRequest и идентификатор пользователя. Выход: dto.GoalResponse.
	ConfirmGoal(ctx context.Context, userID int64, request dto.ConfirmGoalRequest) (dto.GoalResponse, error)
}
