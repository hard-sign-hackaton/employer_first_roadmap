package ports

import (
	"context"

	"efr_bot/models"
)

// EducationOptionsFilter описывает параметры выборки вариантов поступления.
type EducationOptionsFilter struct {
	CareerDirectionID int64
	AdmissionYear     int16
	ExamSubjectIDs    []int64
	RegionID          int64
	ExpandGeography   bool
}

// EducationStore предоставляет сервисам данные об ОП, ЕГЭ и приёмной кампании.
type EducationStore interface {
	ListExamCombinationsForDirection(ctx context.Context, careerDirectionID int64, admissionYear int16) ([]models.ExamCombination, error)
	ListEducationOptions(ctx context.Context, filter EducationOptionsFilter) ([]models.EducationProgram, error)
	ListDirectionIDsAvailableForSubjects(ctx context.Context, companyID int64, examSubjectIDs []int64, admissionYear int16) ([]int64, error)
	GetAdmissionCampaignRule(ctx context.Context, admissionYear int16) (models.AdmissionCampaignRule, error)
}
