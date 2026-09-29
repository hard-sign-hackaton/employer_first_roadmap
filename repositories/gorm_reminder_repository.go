package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"efr_bot/models"
	"efr_bot/reminders"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormReminderRepository reads active roadmap actions and records delivery
// attempts in the same PostgreSQL database as the bot data.
type GormReminderRepository struct {
	db *gorm.DB
}

var _ reminders.Store = (*GormReminderRepository)(nil)

func NewGormReminderRepository(db *gorm.DB) *GormReminderRepository {
	return &GormReminderRepository{db: db}
}

func (r *GormReminderRepository) ListActiveTargets(ctx context.Context) ([]reminders.Target, error) {
	var roadmaps []models.Roadmap
	if err := r.db.WithContext(ctx).
		Preload("UserGoal").
		Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("order_no ASC") }).
		Joins("JOIN user_goals ON user_goals.id = roadmaps.user_goal_id").
		Where("roadmaps.status = ?", models.RoadmapStatusActive).
		Order("roadmaps.id ASC").
		Find(&roadmaps).Error; err != nil {
		return nil, err
	}
	return reminderTargets(roadmaps), nil
}

func (r *GormReminderRepository) FindActiveTargetByUserID(ctx context.Context, userID int64) (reminders.Target, error) {
	var roadmap models.Roadmap
	err := r.db.WithContext(ctx).
		Preload("UserGoal").
		Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("order_no ASC") }).
		Joins("JOIN user_goals ON user_goals.id = roadmaps.user_goal_id").
		Where("user_goals.user_profile_id = ? AND roadmaps.status = ?", userID, models.RoadmapStatusActive).
		Order("roadmaps.created_at DESC").
		First(&roadmap).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return reminders.Target{}, reminders.ErrNoActiveRoadmap
	}
	if err != nil {
		return reminders.Target{}, err
	}
	targets := reminderTargets([]models.Roadmap{roadmap})
	if len(targets) == 0 {
		return reminders.Target{}, reminders.ErrNoActiveRoadmap
	}
	return targets[0], nil
}

func (r *GormReminderRepository) ClaimWeeklyDelivery(ctx context.Context, delivery *models.RoadmapReminderDelivery) (bool, error) {
	if delivery == nil {
		return false, fmt.Errorf("reminder delivery is required")
	}
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "roadmap_id"}, {Name: "week_start"}}, DoNothing: true}).
		Create(delivery)
	return result.RowsAffected == 1, result.Error
}

func (r *GormReminderRepository) FinishWeeklyDelivery(ctx context.Context, deliveryID int64, status string, attempts int16, sentAt *time.Time, sendError string) error {
	return r.db.WithContext(ctx).
		Model(&models.RoadmapReminderDelivery{}).
		Where("id = ?", deliveryID).
		Updates(map[string]any{
			"status":   status,
			"attempts": attempts,
			"sent_at":  sentAt,
			"error":    sendError,
		}).Error
}

func reminderTargets(roadmaps []models.Roadmap) []reminders.Target {
	targets := make([]reminders.Target, 0, len(roadmaps))
	for _, roadmap := range roadmaps {
		for _, step := range roadmap.Steps {
			if step.Status != models.RoadmapStepStatusActive && step.Status != models.RoadmapStepStatusPending {
				continue
			}
			targets = append(targets, reminders.Target{
				UserID:    roadmap.UserGoal.UserProfileID,
				RoadmapID: roadmap.ID,
				StepID:    step.ID,
				StepTitle: step.Title,
			})
			break
		}
	}
	return targets
}
