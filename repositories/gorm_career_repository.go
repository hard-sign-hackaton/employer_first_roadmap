package repositories

import (
	"context"

	"efr_bot/models"
	"efr_bot/services/ports"
	"gorm.io/gorm"
)

// GormCareerRepository реализует CareerRepository через GORM.
type GormCareerRepository struct {
	db *gorm.DB
}

var _ ports.CareerStore = (*GormCareerRepository)(nil)

// NewGormCareerRepository создаёт репозиторий работодателей на переданном подключении GORM.
func NewGormCareerRepository(db *gorm.DB) *GormCareerRepository {
	return &GormCareerRepository{db: db}
}

func (r *GormCareerRepository) FindCompanies(ctx context.Context, query string, limit int) ([]models.Company, error) {
	companiesQuery := r.db.WithContext(ctx).Order("name ASC")
	if query != "" {
		companiesQuery = companiesQuery.Where("LOWER(name) LIKE LOWER(?)", "%"+query+"%")
	}
	if limit > 0 {
		companiesQuery = companiesQuery.Limit(limit)
	}

	var companies []models.Company
	err := companiesQuery.Find(&companies).Error
	return companies, err
}

func (r *GormCareerRepository) FindCompanyByID(ctx context.Context, companyID int64) (models.Company, error) {
	var company models.Company
	err := r.db.WithContext(ctx).First(&company, companyID).Error
	return company, err
}

func (r *GormCareerRepository) ListCompaniesWithDirectionsAndTags(ctx context.Context, limit int) ([]models.Company, error) {
	companiesQuery := r.db.WithContext(ctx).
		Preload("CareerDirections.InterestTags.InterestTag").
		Order("name ASC")
	if limit > 0 {
		companiesQuery = companiesQuery.Limit(limit)
	}

	var companies []models.Company
	err := companiesQuery.Find(&companies).Error
	return companies, err
}

func (r *GormCareerRepository) ListCareerDirectionsByCompany(ctx context.Context, companyID int64) ([]models.CareerDirection, error) {
	var directions []models.CareerDirection
	err := r.db.WithContext(ctx).
		Preload("InterestTags.InterestTag").
		Where("company_id = ?", companyID).
		Order("name ASC").
		Find(&directions).Error
	return directions, err
}

func (r *GormCareerRepository) FindActiveOpportunity(ctx context.Context, companyID, careerDirectionID int64, regionID int64) (models.CompanyOpportunity, error) {
	var opportunity models.CompanyOpportunity
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND career_direction_id = ? AND is_active = ?", companyID, careerDirectionID, true).
		Where("region_id IS NULL OR region_id = ?", regionID).
		Order(gorm.Expr("CASE WHEN region_id = ? THEN 0 ELSE 1 END", regionID)).
		Order("min_study_year ASC").
		First(&opportunity).Error
	return opportunity, err
}
