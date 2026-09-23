package repositories

import (
	"context"

	"efr_bot/models"
)

// EducationOptionsFilter описывает только параметры SQL/GORM-выборки вариантов поступления.
// Проверку фактических баллов и формирование объяснения выполняет сервис, а не репозиторий.
type EducationOptionsFilter struct {
	CareerDirectionID int64
	AdmissionYear     int16
	ExamSubjectIDs    []int64
	RegionID          int64
	ExpandGeography   bool
}

// EducationRepository предоставляет данные об ОП, требованиях ЕГЭ и приёмных кампаниях.
type EducationRepository interface {
	// ListExamCombinationsForDirection возвращает связанные с направлением комбинации ЕГЭ.
	// Вход: идентификатор направления и год приёма. Выход: список models.ExamCombination с ОП и предметами.
	ListExamCombinationsForDirection(ctx context.Context, careerDirectionID int64, admissionYear int16) ([]models.ExamCombination, error)

	// ListEducationOptions получает варианты ОП и вузов по условиям поиска.
	// Вход: EducationOptionsFilter. Выход: список models.EducationProgram с требованиями и вузом.
	ListEducationOptions(ctx context.Context, filter EducationOptionsFilter) ([]models.EducationProgram, error)

	// ListDirectionIDsAvailableForSubjects возвращает направления компании, доступные с выбранными ЕГЭ.
	// Вход: компания, набор предметов и год приёма. Выход: идентификаторы карьерных направлений.
	ListDirectionIDsAvailableForSubjects(ctx context.Context, companyID int64, examSubjectIDs []int64, admissionYear int16) ([]int64, error)

	// GetAdmissionCampaignRule получает лимиты подачи документов для года кампании.
	// Вход: год приёма. Выход: models.AdmissionCampaignRule.
	GetAdmissionCampaignRule(ctx context.Context, admissionYear int16) (models.AdmissionCampaignRule, error)
}
