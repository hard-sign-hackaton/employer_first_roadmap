package reminders

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerConfigFromEnvUsesMoscowMondayDefaults(t *testing.T) {
	t.Setenv("REMINDERS_ENABLED", "")
	t.Setenv("REMINDER_CRON", "")
	t.Setenv("REMINDER_TIMEZONE", "")

	config, err := SchedulerConfigFromEnv()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !config.Enabled || config.Schedule != "0 10 * * 1" || config.Location.String() != "Europe/Moscow" {
		t.Fatalf("config = %#v", config)
	}
	now := time.Date(2026, time.September, 28, 10, 0, 0, 0, config.Location)
	if got := mondayStart(now); got.Weekday() != time.Monday || got.Hour() != 0 {
		t.Fatalf("week start = %s", got)
	}
}

func TestSchedulerConfigRejectsInvalidValues(t *testing.T) {
	t.Setenv("REMINDERS_ENABLED", "sometimes")
	if _, err := SchedulerConfigFromEnv(); err == nil {
		t.Fatal("invalid enabled flag must fail")
	}
	t.Setenv("REMINDERS_ENABLED", "true")
	t.Setenv("REMINDER_CRON", "not a cron expression")
	if _, err := SchedulerConfigFromEnv(); err == nil {
		t.Fatal("invalid cron expression must fail")
	}
}

func TestStartSchedulerDoesNothingWhenDisabled(t *testing.T) {
	scheduler, err := StartScheduler(context.Background(), NewService(&fakeStore{}, &fakeSender{}), SchedulerConfig{Enabled: false})
	if err != nil || scheduler != nil {
		t.Fatalf("disabled scheduler = (%#v, %v), want (nil, nil)", scheduler, err)
	}
}
