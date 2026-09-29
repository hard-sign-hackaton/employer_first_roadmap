package reminders

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	defaultCron     = "0 10 * * 1"
	defaultTimezone = "Europe/Moscow"
)

type SchedulerConfig struct {
	Enabled  bool
	Schedule string
	Location *time.Location
}

func SchedulerConfigFromEnv() (SchedulerConfig, error) {
	enabled := strings.ToLower(strings.TrimSpace(os.Getenv("REMINDERS_ENABLED")))
	if enabled == "" {
		enabled = "true"
	}
	if enabled != "true" && enabled != "false" {
		return SchedulerConfig{}, fmt.Errorf("REMINDERS_ENABLED must be true or false")
	}

	schedule := strings.TrimSpace(os.Getenv("REMINDER_CRON"))
	if schedule == "" {
		schedule = defaultCron
	}
	timezone := strings.TrimSpace(os.Getenv("REMINDER_TIMEZONE"))
	if timezone == "" {
		timezone = defaultTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return SchedulerConfig{}, fmt.Errorf("load reminder timezone: %w", err)
	}
	if _, err := cron.ParseStandard(schedule); err != nil {
		return SchedulerConfig{}, fmt.Errorf("parse REMINDER_CRON: %w", err)
	}
	return SchedulerConfig{Enabled: enabled == "true", Schedule: schedule, Location: location}, nil
}

// StartScheduler runs weekly reminder jobs in the same bot process.
func StartScheduler(ctx context.Context, service *Service, config SchedulerConfig) (*cron.Cron, error) {
	if !config.Enabled {
		return nil, nil
	}
	if service == nil {
		return nil, fmt.Errorf("reminder service is required")
	}
	scheduler := cron.New(cron.WithLocation(config.Location))
	if _, err := scheduler.AddFunc(config.Schedule, func() {
		if err := service.SendWeekly(ctx, time.Now().In(config.Location)); err != nil {
			log.Printf("weekly reminders completed with errors: %v", err)
		}
	}); err != nil {
		return nil, fmt.Errorf("add reminder cron job: %w", err)
	}
	scheduler.Start()
	return scheduler, nil
}
