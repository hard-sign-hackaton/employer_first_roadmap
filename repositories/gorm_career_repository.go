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
	companiesQuery := r.db.WithContext(ctx).Where("is_active = ?", true).Order("name ASC")
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
	err := r.db.WithContext(ctx).Where("is_active = ?", true).First(&company, companyID).Error
	return company, err
}

func (r *GormCareerRepository) ListCompaniesWithDirectionsAndTags(ctx context.Context, limit int) ([]models.Company, error) {
	companiesQuery := r.db.WithContext(ctx).
		Preload("CareerDirections.InterestTags.InterestTag").
		Preload("CareerDirections", "is_active = ?", true).
		Where("companies.is_active = ?", true).
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
		Where("company_id = ? AND is_active = ?", companyID, true).
		Order("name ASC").
		Find(&directions).Error
	return directions, err
}

func (r *GormCareerRepository) FindCareerDirectionByID(ctx context.Context, careerDirectionID int64) (models.CareerDirection, error) {
	var direction models.CareerDirection
	err := r.db.WithContext(ctx).
		Preload("Company").
		Preload("InterestTags.InterestTag").
		Where("career_directions.is_active = ?", true).
		First(&direction, careerDirectionID).Error
	return direction, err
}

func (r *GormCareerRepository) FindActiveOpportunity(ctx context.Context, companyID, careerDirectionID int64, regionID int64) (models.CompanyOpportunity, error) {
	var opportunity models.CompanyOpportunity
	err := r.db.WithContext(ctx).
		Where("company_id = ? AND career_direction_id = ? AND is_active = ?", companyID, careerDirectionID, true).
		// A region-less opportunity is not automatically nationwide: only a
		// source-confirmed remote format may be offered outside the university's region.
		Where("region_id = ? OR (region_id IS NULL AND work_format = 'remote')", regionID).
		Order(gorm.Expr("CASE WHEN region_id = ? THEN 0 ELSE 1 END", regionID)).
		Order("min_study_year ASC").
		First(&opportunity).Error
	return opportunity, err
}

func (r *GormCareerRepository) ListActiveOpportunities(ctx context.Context, companyID, careerDirectionID, regionID int64, excludedIDs []int64) ([]models.CompanyOpportunity, error) {
	var opportunities []models.CompanyOpportunity
	query := r.db.WithContext(ctx).
		Preload("Company").
		Where("company_id = ? AND career_direction_id = ? AND is_active = ?", companyID, careerDirectionID, true).
		Where("region_id = ? OR (region_id IS NULL AND work_format = 'remote')", regionID).
		Order(gorm.Expr("CASE WHEN region_id = ? THEN 0 ELSE 1 END", regionID)).
		Order("min_study_year ASC, id ASC")
	if len(excludedIDs) > 0 {
		query = query.Where("id NOT IN ?", excludedIDs)
	}
	if err := query.Find(&opportunities).Error; err != nil {
		return nil, err
	}
	return opportunities, nil
}
