package ports

import (
	"context"

	"efr_bot/models"
)

// EducationOptionsFilter описывает параметры выборки вариантов поступления.
type EducationOptionsFilter struct {
	CareerDirectionID int64
	ExamSubjectIDs    []int64
	RegionID          int64
	ExpandGeography   bool
}

// EducationStore предоставляет сервисам данные об ОП, ЕГЭ и приёмной кампании.
type EducationStore interface {
	// ListLatestExamCombinationsForDirection возвращает последний опубликованный
	// срез комбинаций ЕГЭ, имеющийся в каталоге.
	ListLatestExamCombinationsForDirection(ctx context.Context, careerDirectionID int64) ([]models.ExamCombination, error)
	ListEducationOptions(ctx context.Context, filter EducationOptionsFilter) ([]models.EducationProgram, error)
	ListDirectionIDsAvailableForSubjects(ctx context.Context, companyID int64, examSubjectIDs []int64) ([]int64, error)
	// GetLatestAdmissionCampaignRule возвращает последнее правило из каталога.
	GetLatestAdmissionCampaignRule(ctx context.Context) (models.AdmissionCampaignRule, error)
}
