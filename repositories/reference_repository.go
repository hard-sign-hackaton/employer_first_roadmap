package repositories

import (
	"context"

	"efr_bot/models"
)

// ReferenceRepository предоставляет справочники, используемые в вопросах бота.
type ReferenceRepository interface {
	// ListRegions возвращает регионы, доступные для выбора в профиле.
	// Вход: контекст. Выход: список models.Region.
	ListRegions(ctx context.Context) ([]models.Region, error)

	// ListInterestTags возвращает теги для нормализации ответов полного опроса.
	// Вход: контекст. Выход: список models.InterestTag.
	ListInterestTags(ctx context.Context) ([]models.InterestTag, error)

	// ListExamSubjects возвращает предметы ЕГЭ для выбора и фиксации результатов.
	// Вход: контекст. Выход: список models.ExamSubject.
	ListExamSubjects(ctx context.Context) ([]models.ExamSubject, error)
}
