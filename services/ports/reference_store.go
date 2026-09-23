package ports

import (
	"context"

	"efr_bot/models"
)

// ReferenceStore предоставляет справочники для вопросов бота.
type ReferenceStore interface {
	ListRegions(ctx context.Context) ([]models.Region, error)
	ListInterestTags(ctx context.Context) ([]models.InterestTag, error)
	ListExamSubjects(ctx context.Context) ([]models.ExamSubject, error)
}
