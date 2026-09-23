package repositories

import (
	"context"

	"efr_bot/models"
)

// ProfileRepository хранит минимальный профиль, интересы и предметы ЕГЭ пользователя.
type ProfileRepository interface {
	// FindProfileByUserID получает профиль пользователя вместе с регионом.
	// Вход: идентификатор пользователя MAX. Выход: models.UserProfile.
	FindProfileByUserID(ctx context.Context, userID int64) (models.UserProfile, error)

	// SaveProfile создаёт или обновляет минимальный профиль.
	// Вход: models.UserProfile. Выход: сохранённый models.UserProfile.
	SaveProfile(ctx context.Context, profile models.UserProfile) (models.UserProfile, error)

	// ReplaceUserInterests полностью заменяет результат полного опроса пользователя.
	// Вход: идентификатор пользователя и список models.UserInterest. Выход: сохранённый список.
	ReplaceUserInterests(ctx context.Context, userID int64, interests []models.UserInterest) ([]models.UserInterest, error)

	// ListUserInterests возвращает интересы пользователя вместе с данными тегов.
	// Вход: идентификатор пользователя. Выход: список models.UserInterest.
	ListUserInterests(ctx context.Context, userID int64) ([]models.UserInterest, error)

	// ReplaceUserSubjects полностью заменяет планируемые или выбранные предметы ЕГЭ.
	// Вход: идентификатор пользователя и список models.UserSubject. Выход: сохранённый список.
	ReplaceUserSubjects(ctx context.Context, userID int64, subjects []models.UserSubject) ([]models.UserSubject, error)

	// ListUserSubjects возвращает предметы ЕГЭ пользователя вместе с названиями предметов.
	// Вход: идентификатор пользователя. Выход: список models.UserSubject.
	ListUserSubjects(ctx context.Context, userID int64) ([]models.UserSubject, error)

	// SaveExamResults сохраняет фактические баллы и переводит предметы в статус passed.
	// Вход: идентификатор пользователя и список models.UserSubject. Выход: обновлённый список.
	SaveExamResults(ctx context.Context, userID int64, subjects []models.UserSubject) ([]models.UserSubject, error)
}
