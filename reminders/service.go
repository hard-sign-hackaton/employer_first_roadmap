// Package reminders sends periodic MAX messages for active roadmaps.
package reminders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"efr_bot/models"
)

var ErrNoActiveRoadmap = errors.New("no active roadmap")

const maxSendAttempts int16 = 3

// Target is an active roadmap and its next action.
type Target struct {
	UserID    int64
	RoadmapID int64
	StepID    int64
	StepTitle string
}

// Store is the persistence boundary for weekly delivery idempotency.
type Store interface {
	ListActiveTargets(ctx context.Context) ([]Target, error)
	FindActiveTargetByUserID(ctx context.Context, userID int64) (Target, error)
	ClaimWeeklyDelivery(ctx context.Context, delivery *models.RoadmapReminderDelivery) (bool, error)
	FinishWeeklyDelivery(ctx context.Context, deliveryID int64, status string, attempts int16, sentAt *time.Time, sendError string) error
}

// Sender delivers an already rendered reminder to one MAX user.
type Sender interface {
	Send(ctx context.Context, userID int64, text string) error
}

type Service struct {
	store  Store
	sender Sender
}

func NewService(store Store, sender Sender) *Service {
	return &Service{store: store, sender: sender}
}

// SendWeekly sends one protected reminder per active roadmap for the week that
// contains scheduledAt. A failure for one user never stops other deliveries.
func (s *Service) SendWeekly(ctx context.Context, scheduledAt time.Time) error {
	targets, err := s.store.ListActiveTargets(ctx)
	if err != nil {
		return fmt.Errorf("list reminder targets: %w", err)
	}

	weekStart := mondayStart(scheduledAt)
	var failures []error
	for _, target := range targets {
		delivery := models.RoadmapReminderDelivery{
			RoadmapID: target.RoadmapID,
			UserID:    target.UserID,
			WeekStart: weekStart,
			Status:    models.ReminderDeliveryStatusPending,
		}
		claimed, claimErr := s.store.ClaimWeeklyDelivery(ctx, &delivery)
		if claimErr != nil {
			failures = append(failures, fmt.Errorf("claim reminder for roadmap %d: %w", target.RoadmapID, claimErr))
			continue
		}
		if !claimed {
			continue
		}

		attempts, sendErr := s.send(ctx, target)
		if sendErr != nil {
			if err := s.store.FinishWeeklyDelivery(ctx, delivery.ID, models.ReminderDeliveryStatusFailed, attempts, nil, sendErr.Error()); err != nil {
				failures = append(failures, fmt.Errorf("record failed reminder for roadmap %d: %w", target.RoadmapID, err))
			} else {
				failures = append(failures, fmt.Errorf("send reminder for roadmap %d: %w", target.RoadmapID, sendErr))
			}
			continue
		}

		sentAt := time.Now().UTC()
		if err := s.store.FinishWeeklyDelivery(ctx, delivery.ID, models.ReminderDeliveryStatusSent, attempts, &sentAt, ""); err != nil {
			failures = append(failures, fmt.Errorf("record sent reminder for roadmap %d: %w", target.RoadmapID, err))
		}
	}
	return errors.Join(failures...)
}

// SendTestReminder deliberately bypasses the weekly delivery journal so a user
// can verify a real MAX delivery at any time.
func (s *Service) SendTestReminder(ctx context.Context, userID int64) error {
	target, err := s.store.FindActiveTargetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	_, err = s.send(ctx, target)
	return err
}

func (s *Service) send(ctx context.Context, target Target) (int16, error) {
	var lastErr error
	for attempt := int16(1); attempt <= maxSendAttempts; attempt++ {
		if err := s.sender.Send(ctx, target.UserID, ReminderText(target)); err == nil {
			return attempt, nil
		} else {
			lastErr = err
		}
	}
	return maxSendAttempts, lastErr
}

func ReminderText(target Target) string {
	return "Напоминаем: ваша текущая задача — «" + strings.TrimSpace(target.StepTitle) + "».\n\nНе забывайте работать над ней. Если уже выполнили задачу — откройте roadmap и отметьте шаг."
}

func mondayStart(at time.Time) time.Time {
	local := at
	dayOffset := (int(local.Weekday()) + 6) % 7
	return time.Date(local.Year(), local.Month(), local.Day()-dayOffset, 0, 0, 0, 0, local.Location())
}
