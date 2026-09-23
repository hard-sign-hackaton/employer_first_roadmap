package repositories

import (
	"context"

	"efr_bot/models"
	"efr_bot/services/ports"
	"gorm.io/gorm"
)

// GormProfileRepository реализует ProfileRepository через GORM.
type GormProfileRepository struct {
	db *gorm.DB
}

var _ ports.ProfileStore = (*GormProfileRepository)(nil)

// NewGormProfileRepository создаёт репозиторий профиля на переданном подключении GORM.
func NewGormProfileRepository(db *gorm.DB) *GormProfileRepository {
	return &GormProfileRepository{db: db}
}

func (r *GormProfileRepository) FindProfileByUserID(ctx context.Context, userID int64) (models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.WithContext(ctx).
		Preload("Region").
		Where("id = ?", userID).
		First(&profile).Error
	return profile, err
}

func (r *GormProfileRepository) SaveProfile(ctx context.Context, profile models.UserProfile) (models.UserProfile, error) {
	err := r.db.WithContext(ctx).Save(&profile).Error
	return profile, err
}

func (r *GormProfileRepository) ReplaceUserInterests(ctx context.Context, userID int64, interests []models.UserInterest) ([]models.UserInterest, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_profile_id = ?", userID).Delete(&models.UserInterest{}).Error; err != nil {
			return err
		}

		for index := range interests {
			interests[index].UserProfileID = userID
		}
		if len(interests) == 0 {
			return nil
		}
		return tx.Create(&interests).Error
	})
	if err != nil {
		return nil, err
	}
	return r.ListUserInterests(ctx, userID)
}

func (r *GormProfileRepository) ListUserInterests(ctx context.Context, userID int64) ([]models.UserInterest, error) {
	var interests []models.UserInterest
	err := r.db.WithContext(ctx).
		Preload("InterestTag").
		Where("user_profile_id = ?", userID).
		Order("interest_tag_id ASC").
		Find(&interests).Error
	return interests, err
}

func (r *GormProfileRepository) ReplaceUserSubjects(ctx context.Context, userID int64, subjects []models.UserSubject) ([]models.UserSubject, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_profile_id = ?", userID).Delete(&models.UserSubject{}).Error; err != nil {
			return err
		}

		for index := range subjects {
			subjects[index].UserProfileID = userID
		}
		if len(subjects) == 0 {
			return nil
		}
		return tx.Create(&subjects).Error
	})
	if err != nil {
		return nil, err
	}
	return r.ListUserSubjects(ctx, userID)
}

func (r *GormProfileRepository) ListUserSubjects(ctx context.Context, userID int64) ([]models.UserSubject, error) {
	var subjects []models.UserSubject
	err := r.db.WithContext(ctx).
		Preload("ExamSubject").
		Where("user_profile_id = ?", userID).
		Order("exam_subject_id ASC").
		Find(&subjects).Error
	return subjects, err
}

func (r *GormProfileRepository) SaveExamResults(ctx context.Context, userID int64, subjects []models.UserSubject) ([]models.UserSubject, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index := range subjects {
			subjects[index].UserProfileID = userID
			subjects[index].Status = models.SubjectStatusPassed
			if err := tx.Save(&subjects[index]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListUserSubjects(ctx, userID)
}
