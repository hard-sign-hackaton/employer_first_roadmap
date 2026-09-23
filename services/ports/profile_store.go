package ports

import (
	"context"

	"efr_bot/models"
)

// ProfileStore предоставляет сервисам доступ к профилю, интересам и предметам ЕГЭ пользователя.
type ProfileStore interface {
	FindProfileByUserID(ctx context.Context, userID int64) (models.UserProfile, error)
	SaveProfile(ctx context.Context, profile models.UserProfile) (models.UserProfile, error)
	ReplaceUserInterests(ctx context.Context, userID int64, interests []models.UserInterest) ([]models.UserInterest, error)
	ListUserInterests(ctx context.Context, userID int64) ([]models.UserInterest, error)
	ReplaceUserSubjects(ctx context.Context, userID int64, subjects []models.UserSubject) ([]models.UserSubject, error)
	ListUserSubjects(ctx context.Context, userID int64) ([]models.UserSubject, error)
	SaveExamResults(ctx context.Context, userID int64, subjects []models.UserSubject) ([]models.UserSubject, error)
}
