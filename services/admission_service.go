package services

import (
	"context"

	"efr_bot/dto"
)

// AdmissionService уточняет roadmap после ЕГЭ: подбирает ОП, сохраняет план подачи
// и фиксирует единственный итоговый выбор после приёмной кампании.
type AdmissionService interface {
	// FindEducationOptions подбирает доступные ОП и вузы по фактическим ЕГЭ, баллам и географии.
	// Вход: dto.FindEducationOptionsRequest и идентификатор пользователя. Выход: список dto.EducationOptionResponse.
	FindEducationOptions(ctx context.Context, userID int64, request dto.FindEducationOptionsRequest) ([]dto.EducationOptionResponse, error)

	// GetAdmissionPlanLimits возвращает действующие ограничения кампании для roadmap.
	// Вход: dto.GetAdmissionPlanRequest и идентификатор пользователя. Выход: dto.AdmissionPlanLimitsResponse.
	GetAdmissionPlanLimits(ctx context.Context, userID int64, request dto.GetAdmissionPlanRequest) (dto.AdmissionPlanLimitsResponse, error)

	// SaveAdmissionPlan сохраняет до допустимого лимита вариантов подачи документов.
	// Вход: dto.GetAdmissionPlanRequest и dto.SaveAdmissionPlanRequest. Выход: список dto.AdmissionApplicationResponse.
	SaveAdmissionPlan(ctx context.Context, userID int64, plan dto.GetAdmissionPlanRequest, request dto.SaveAdmissionPlanRequest) ([]dto.AdmissionApplicationResponse, error)

	// GetAdmissionPlan возвращает ранее сохранённый план подачи документов.
	// Вход: dto.GetAdmissionPlanRequest и идентификатор пользователя. Выход: список dto.AdmissionApplicationResponse.
	GetAdmissionPlan(ctx context.Context, userID int64, request dto.GetAdmissionPlanRequest) ([]dto.AdmissionApplicationResponse, error)

	// SaveEnrollmentChoice фиксирует итоговый выбранный вуз и ОП либо отсутствие поступления.
	// При успешном выборе уточняет будущие шаги roadmap возможностью работодателя.
	// Вход: dto.GetAdmissionPlanRequest и dto.SaveEnrollmentChoiceRequest. Выход: dto.EnrollmentChoiceResponse.
	SaveEnrollmentChoice(ctx context.Context, userID int64, plan dto.GetAdmissionPlanRequest, request dto.SaveEnrollmentChoiceRequest) (dto.EnrollmentChoiceResponse, error)
}
