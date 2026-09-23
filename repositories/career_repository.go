package repositories

import (
	"context"

	"efr_bot/models"
)

// CareerRepository предоставляет работодателей, направления и возможности практического опыта.
type CareerRepository interface {
	// FindCompanies ищет работодателей по части названия.
	// Вход: поисковая строка и лимит. Выход: список models.Company.
	FindCompanies(ctx context.Context, query string, limit int) ([]models.Company, error)

	// FindCompanyByID получает одного работодателя.
	// Вход: идентификатор компании. Выход: models.Company.
	FindCompanyByID(ctx context.Context, companyID int64) (models.Company, error)

	// ListCompaniesWithDirectionsAndTags получает данные для детерминированного ранжирования.
	// Вход: лимит результата. Выход: компании с направлениями и тегами интересов.
	ListCompaniesWithDirectionsAndTags(ctx context.Context, limit int) ([]models.Company, error)

	// ListCareerDirectionsByCompany получает все направления выбранного работодателя с тегами.
	// Вход: идентификатор компании. Выход: список models.CareerDirection.
	ListCareerDirectionsByCompany(ctx context.Context, companyID int64) ([]models.CareerDirection, error)

	// FindActiveOpportunity получает вручную настроенную возможность работодателя для направления.
	// Вход: идентификаторы компании и направления, а также регион пользователя. Выход: models.CompanyOpportunity.
	FindActiveOpportunity(ctx context.Context, companyID, careerDirectionID int64, regionID int64) (models.CompanyOpportunity, error)
}
