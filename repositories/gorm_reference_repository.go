package repositories

import (
	"context"

	"efr_bot/models"
	"efr_bot/services/ports"
	"gorm.io/gorm"
)

// GormReferenceRepository реализует ReferenceRepository через GORM.
type GormReferenceRepository struct {
	db *gorm.DB
}

var _ ports.ReferenceStore = (*GormReferenceRepository)(nil)

// NewGormReferenceRepository создаёт репозиторий справочников на переданном подключении GORM.
func NewGormReferenceRepository(db *gorm.DB) *GormReferenceRepository {
	return &GormReferenceRepository{db: db}
}

func (r *GormReferenceRepository) ListRegions(ctx context.Context) ([]models.Region, error) {
	var regions []models.Region
	err := r.db.WithContext(ctx).Order("name ASC").Find(&regions).Error
	return regions, err
}

func (r *GormReferenceRepository) ListInterestTags(ctx context.Context) ([]models.InterestTag, error) {
	var tags []models.InterestTag
	err := r.db.WithContext(ctx).Order("name ASC").Find(&tags).Error
	return tags, err
}

func (r *GormReferenceRepository) ListExamSubjects(ctx context.Context) ([]models.ExamSubject, error) {
	var subjects []models.ExamSubject
	err := r.db.WithContext(ctx).Order("name ASC").Find(&subjects).Error
	return subjects, err
}
