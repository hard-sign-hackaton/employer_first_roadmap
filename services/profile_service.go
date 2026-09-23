package services

import (
	"context"

	"efr_bot/dto"
)

// ProfileService отвечает за минимальный профиль, интересы и предметы ЕГЭ пользователя.
type ProfileService interface {
	// SaveProfile сохраняет класс, регион и готовность к переезду.
	// Вход: dto.UpsertProfileRequest. Выход: dto.ProfileResponse.
	SaveProfile(ctx context.Context, userID int64, request dto.UpsertProfileRequest) (dto.ProfileResponse, error)

	// GetProfile возвращает сохранённый минимальный профиль пользователя.
	// Вход: идентификатор пользователя из контекста MAX. Выход: dto.ProfileResponse.
	GetProfile(ctx context.Context, userID int64) (dto.ProfileResponse, error)

	// SaveSurveyInterests сохраняет результат полного профориентационного опроса.
	// Вход: dto.SaveSurveyInterestsRequest. Выход: список dto.InterestResponse.
	SaveSurveyInterests(ctx context.Context, userID int64, request dto.SaveSurveyInterestsRequest) ([]dto.InterestResponse, error)

	// SaveUserSubjects сохраняет планируемые или выбранные предметы ЕГЭ.
	// Вход: dto.SaveUserSubjectsRequest. Выход: список dto.UserSubjectResponse.
	SaveUserSubjects(ctx context.Context, userID int64, request dto.SaveUserSubjectsRequest) ([]dto.UserSubjectResponse, error)

	// SaveExamResults фиксирует фактические баллы после сдачи ЕГЭ.
	// Вход: dto.SaveExamResultsRequest. Выход: список dto.UserSubjectResponse со статусом passed.
	SaveExamResults(ctx context.Context, userID int64, request dto.SaveExamResultsRequest) ([]dto.UserSubjectResponse, error)
}
