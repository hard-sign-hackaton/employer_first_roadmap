package repositories

import (
	"context"
	"errors"
	"time"

	"efr_bot/models"
	"efr_bot/services/ports"
	"gorm.io/gorm"
)

// GormRoadmapRepository реализует RoadmapRepository через GORM.
type GormRoadmapRepository struct {
	db *gorm.DB
}

var _ ports.RoadmapStore = (*GormRoadmapRepository)(nil)

// NewGormRoadmapRepository создаёт репозиторий roadmap на переданном подключении GORM.
func NewGormRoadmapRepository(db *gorm.DB) *GormRoadmapRepository {
	return &GormRoadmapRepository{db: db}
}

func (r *GormRoadmapRepository) CreateGoal(ctx context.Context, goal models.UserGoal) (models.UserGoal, error) {
	err := r.db.WithContext(ctx).Create(&goal).Error
	return goal, err
}

func (r *GormRoadmapRepository) FindGoalByID(ctx context.Context, userID, goalID int64) (models.UserGoal, error) {
	var goal models.UserGoal
	err := r.db.WithContext(ctx).
		Preload("CareerDirection.Company").
		Where("id = ? AND user_profile_id = ?", goalID, userID).
		First(&goal).Error
	return goal, err
}

func (r *GormRoadmapRepository) FindActiveRoadmapByUserID(ctx context.Context, userID int64) (models.Roadmap, error) {
	var roadmap models.Roadmap
	err := r.roadmapDetails(r.db.WithContext(ctx)).
		Joins("JOIN user_goals ON user_goals.id = roadmaps.user_goal_id").
		Where("user_goals.user_profile_id = ? AND roadmaps.status = ?", userID, models.RoadmapStatusActive).
		Order("roadmaps.created_at DESC").
		First(&roadmap).Error
	return roadmap, err
}

func (r *GormRoadmapRepository) FindRoadmapByID(ctx context.Context, userID, roadmapID int64) (models.Roadmap, error) {
	var roadmap models.Roadmap
	err := r.roadmapDetails(r.db.WithContext(ctx)).
		Joins("JOIN user_goals ON user_goals.id = roadmaps.user_goal_id").
		Where("roadmaps.id = ? AND user_goals.user_profile_id = ?", roadmapID, userID).
		First(&roadmap).Error
	return roadmap, err
}

func (r *GormRoadmapRepository) FindActiveTemplateForDirection(ctx context.Context, careerDirectionID int64) (models.RoadmapTemplate, error) {
	var template models.RoadmapTemplate
	err := r.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_no ASC")
		}).
		Where("career_direction_id = ? AND is_active = ?", careerDirectionID, true).
		Order("version DESC").
		First(&template).Error
	return template, err
}

func (r *GormRoadmapRepository) CreateRoadmapWithSteps(ctx context.Context, roadmap models.Roadmap, steps []models.RoadmapStep) (models.Roadmap, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&roadmap).Error; err != nil {
			return err
		}

		for index := range steps {
			steps[index].RoadmapID = roadmap.ID
		}
		if len(steps) == 0 {
			return nil
		}
		return tx.Create(&steps).Error
	})
	if err != nil {
		return models.Roadmap{}, err
	}

	return r.findRoadmapDetails(ctx, roadmap.ID)
}

func (r *GormRoadmapRepository) UpdateRoadmapStepStatus(ctx context.Context, roadmapID, stepID int64, status string) (models.RoadmapStep, error) {
	updates := map[string]any{"status": status}
	if status == models.RoadmapStepStatusCompleted {
		updates["completed_at"] = time.Now().UTC()
	}

	if err := r.db.WithContext(ctx).
		Model(&models.RoadmapStep{}).
		Where("id = ? AND roadmap_id = ?", stepID, roadmapID).
		Updates(updates).Error; err != nil {
		return models.RoadmapStep{}, err
	}

	var step models.RoadmapStep
	err := r.db.WithContext(ctx).
		Preload("CompanyOpportunity").
		Where("id = ? AND roadmap_id = ?", stepID, roadmapID).
		First(&step).Error
	return step, err
}

func (r *GormRoadmapRepository) CompleteRoadmapStep(ctx context.Context, roadmapID, stepID int64) (models.RoadmapStep, error) {
	completedAt := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var currentStep models.RoadmapStep
		if err := tx.
			Where("id = ? AND roadmap_id = ? AND status = ?", stepID, roadmapID, models.RoadmapStepStatusActive).
			First(&currentStep).Error; err != nil {
			return err
		}
		if err := tx.Model(&currentStep).Updates(map[string]any{
			"status":       models.RoadmapStepStatusCompleted,
			"completed_at": completedAt,
		}).Error; err != nil {
			return err
		}

		var nextStep models.RoadmapStep
		err := tx.
			Where("roadmap_id = ? AND status = ? AND order_no > ?", roadmapID, models.RoadmapStepStatusPending, currentStep.OrderNo).
			Order("order_no ASC").
			First(&nextStep).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Model(&models.Roadmap{}).
				Where("id = ?", roadmapID).
				Updates(map[string]any{
					"status":       models.RoadmapStatusCompleted,
					"completed_at": completedAt,
				}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&nextStep).Update("status", models.RoadmapStepStatusActive).Error
	})
	if err != nil {
		return models.RoadmapStep{}, err
	}

	var completedStep models.RoadmapStep
	err = r.db.WithContext(ctx).
		Preload("CompanyOpportunity").
		Where("id = ? AND roadmap_id = ?", stepID, roadmapID).
		First(&completedStep).Error
	return completedStep, err
}

func (r *GormRoadmapRepository) ReplaceUncompletedSteps(ctx context.Context, roadmapID int64, steps []models.RoadmapStep) ([]models.RoadmapStep, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("roadmap_id = ? AND status IN ?", roadmapID, []string{models.RoadmapStepStatusPending, models.RoadmapStepStatusActive}).
			Delete(&models.RoadmapStep{}).Error; err != nil {
			return err
		}

		for index := range steps {
			steps[index].RoadmapID = roadmapID
		}
		if len(steps) == 0 {
			return nil
		}
		return tx.Create(&steps).Error
	})
	if err != nil {
		return nil, err
	}

	var savedSteps []models.RoadmapStep
	err = r.db.WithContext(ctx).
		Where("roadmap_id = ?", roadmapID).
		Order("order_no ASC").
		Find(&savedSteps).Error
	return savedSteps, err
}

func (r *GormRoadmapRepository) ReplaceAdmissionApplications(ctx context.Context, roadmapID int64, applications []models.RoadmapAdmissionApplication) ([]models.RoadmapAdmissionApplication, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("roadmap_id = ?", roadmapID).Delete(&models.RoadmapAdmissionApplication{}).Error; err != nil {
			return err
		}

		for index := range applications {
			applications[index].RoadmapID = roadmapID
		}
		if len(applications) == 0 {
			return nil
		}
		return tx.Create(&applications).Error
	})
	if err != nil {
		return nil, err
	}
	return r.ListAdmissionApplications(ctx, roadmapID)
}

func (r *GormRoadmapRepository) ListAdmissionApplications(ctx context.Context, roadmapID int64) ([]models.RoadmapAdmissionApplication, error) {
	var applications []models.RoadmapAdmissionApplication
	err := r.db.WithContext(ctx).
		Preload("EducationProgram.University").
		Where("roadmap_id = ?", roadmapID).
		Order("id ASC").
		Find(&applications).Error
	return applications, err
}

func (r *GormRoadmapRepository) SaveEnrollmentChoice(ctx context.Context, choice models.RoadmapEnrollmentChoice) (models.RoadmapEnrollmentChoice, error) {
	err := r.db.WithContext(ctx).Save(&choice).Error
	return choice, err
}

func (r *GormRoadmapRepository) SaveEmployerApplication(ctx context.Context, application models.RoadmapEmployerApplication) (models.RoadmapEmployerApplication, error) {
	err := r.db.WithContext(ctx).Save(&application).Error
	return application, err
}

func (r *GormRoadmapRepository) ArchiveRoadmap(ctx context.Context, roadmapID int64) (models.Roadmap, error) {
	archivedAt := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Model(&models.Roadmap{}).
		Where("id = ?", roadmapID).
		Updates(map[string]any{
			"status":      models.RoadmapStatusArchived,
			"archived_at": archivedAt,
		}).Error; err != nil {
		return models.Roadmap{}, err
	}

	return r.findRoadmapDetails(ctx, roadmapID)
}

func (r *GormRoadmapRepository) findRoadmapDetails(ctx context.Context, roadmapID int64) (models.Roadmap, error) {
	var roadmap models.Roadmap
	err := r.roadmapDetails(r.db.WithContext(ctx)).First(&roadmap, roadmapID).Error
	return roadmap, err
}

func (r *GormRoadmapRepository) roadmapDetails(query *gorm.DB) *gorm.DB {
	return query.
		Preload("UserGoal.CareerDirection.Company").
		Preload("RoadmapTemplate").
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_no ASC")
		}).
		Preload("Steps.CompanyOpportunity").
		Preload("AdmissionApplications.EducationProgram.University").
		Preload("EnrollmentChoice.AdmissionApplication.EducationProgram.University").
		Preload("EmployerApplication.CompanyOpportunity")
}
