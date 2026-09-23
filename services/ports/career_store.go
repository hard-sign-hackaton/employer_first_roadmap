package ports

import (
	"context"

	"efr_bot/models"
)

// CareerStore предоставляет сервисам работодателей, направления и возможности практического опыта.
type CareerStore interface {
	FindCompanies(ctx context.Context, query string, limit int) ([]models.Company, error)
	FindCompanyByID(ctx context.Context, companyID int64) (models.Company, error)
	ListCompaniesWithDirectionsAndTags(ctx context.Context, limit int) ([]models.Company, error)
	ListCareerDirectionsByCompany(ctx context.Context, companyID int64) ([]models.CareerDirection, error)
	FindActiveOpportunity(ctx context.Context, companyID, careerDirectionID int64, regionID int64) (models.CompanyOpportunity, error)
}
