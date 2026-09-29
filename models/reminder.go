package models

import "time"

const (
	ReminderDeliveryStatusPending = "pending"
	ReminderDeliveryStatusSent    = "sent"
	ReminderDeliveryStatusFailed  = "failed"
)

// RoadmapReminderDelivery keeps the result of one weekly reminder attempt.
// WeekStart is always the Monday of the configured scheduler timezone.
type RoadmapReminderDelivery struct {
	ID        int64      `gorm:"primaryKey;autoIncrement"`
	RoadmapID int64      `gorm:"not null;uniqueIndex:ux_roadmap_reminder_week"`
	UserID    int64      `gorm:"not null;index"`
	WeekStart time.Time  `gorm:"not null;uniqueIndex:ux_roadmap_reminder_week"`
	Status    string     `gorm:"size:16;not null;check:status IN ('pending','sent','failed')"`
	Attempts  int16      `gorm:"not null;default:0;check:attempts >= 0"`
	SentAt    *time.Time `gorm:"default:null"`
	Error     string     `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
