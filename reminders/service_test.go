package reminders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"efr_bot/models"
)

func TestSendWeeklySendsOnlyOncePerRoadmapAndWeek(t *testing.T) {
	store := &fakeStore{targets: []Target{
		{UserID: 101, RoadmapID: 10, StepID: 1, StepTitle: "Подготовиться к ЕГЭ"},
		{UserID: 202, RoadmapID: 20, StepID: 2, StepTitle: "Подать документы"},
	}}
	sender := &fakeSender{}
	service := NewService(store, sender)
	now := time.Date(2026, time.September, 28, 10, 0, 0, 0, time.FixedZone("MSK", 3*60*60))

	if err := service.SendWeekly(context.Background(), now); err != nil {
		t.Fatalf("first weekly send: %v", err)
	}
	if err := service.SendWeekly(context.Background(), now.Add(2*time.Hour)); err != nil {
		t.Fatalf("duplicate weekly send: %v", err)
	}

	if got := len(sender.messages); got != 2 {
		t.Fatalf("messages = %d, want 2", got)
	}
	for _, delivery := range store.deliveries {
		if delivery.Status != models.ReminderDeliveryStatusSent || delivery.Attempts != 1 || delivery.SentAt == nil {
			t.Fatalf("delivery = %#v, want sent after one attempt", delivery)
		}
	}
	if !strings.Contains(sender.messages[0].text, "Подготовиться к ЕГЭ") {
		t.Fatalf("reminder text = %q", sender.messages[0].text)
	}
}

func TestSendWeeklyRetriesFailuresAndContinuesWithOtherUsers(t *testing.T) {
	store := &fakeStore{targets: []Target{
		{UserID: 101, RoadmapID: 10, StepTitle: "Первый шаг"},
		{UserID: 202, RoadmapID: 20, StepTitle: "Второй шаг"},
	}}
	sender := &fakeSender{failures: map[int64]int{101: 3}}
	service := NewService(store, sender)

	err := service.SendWeekly(context.Background(), time.Date(2026, time.September, 28, 10, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("weekly send must report a failed recipient")
	}
	if got := sender.calls[101]; got != 3 {
		t.Fatalf("failed recipient calls = %d, want 3", got)
	}
	if got := sender.calls[202]; got != 1 {
		t.Fatalf("successful recipient calls = %d, want 1", got)
	}
	if store.deliveries[0].Status != models.ReminderDeliveryStatusFailed || store.deliveries[0].Attempts != 3 {
		t.Fatalf("failed delivery = %#v", store.deliveries[0])
	}
	if store.deliveries[1].Status != models.ReminderDeliveryStatusSent {
		t.Fatalf("successful delivery = %#v", store.deliveries[1])
	}
}

func TestSendTestReminderDoesNotCreateWeeklyDelivery(t *testing.T) {
	store := &fakeStore{targets: []Target{{UserID: 101, RoadmapID: 10, StepTitle: "Выбрать вуз"}}}
	sender := &fakeSender{}
	service := NewService(store, sender)

	if err := service.SendTestReminder(context.Background(), 101); err != nil {
		t.Fatalf("send test reminder: %v", err)
	}
	if len(store.deliveries) != 0 {
		t.Fatalf("manual reminder must not create weekly delivery: %#v", store.deliveries)
	}
	if got := sender.calls[101]; got != 1 {
		t.Fatalf("manual reminder calls = %d, want 1", got)
	}
	if err := service.SendTestReminder(context.Background(), 404); !errors.Is(err, ErrNoActiveRoadmap) {
		t.Fatalf("missing roadmap error = %v, want ErrNoActiveRoadmap", err)
	}
}

func TestReminderTextIncludesCurrentStep(t *testing.T) {
	text := ReminderText(Target{StepTitle: "  Подтвердить зачисление  "})
	if !strings.Contains(text, "«Подтвердить зачисление»") || !strings.Contains(text, "откройте roadmap") {
		t.Fatalf("text = %q", text)
	}
}

type fakeStore struct {
	targets    []Target
	deliveries []models.RoadmapReminderDelivery
	nextID     int64
}

func (s *fakeStore) ListActiveTargets(context.Context) ([]Target, error) {
	return append([]Target(nil), s.targets...), nil
}

func (s *fakeStore) FindActiveTargetByUserID(_ context.Context, userID int64) (Target, error) {
	for _, target := range s.targets {
		if target.UserID == userID {
			return target, nil
		}
	}
	return Target{}, ErrNoActiveRoadmap
}

func (s *fakeStore) ClaimWeeklyDelivery(_ context.Context, delivery *models.RoadmapReminderDelivery) (bool, error) {
	for _, current := range s.deliveries {
		if current.RoadmapID == delivery.RoadmapID && current.WeekStart.Equal(delivery.WeekStart) {
			return false, nil
		}
	}
	s.nextID++
	delivery.ID = s.nextID
	s.deliveries = append(s.deliveries, *delivery)
	return true, nil
}

func (s *fakeStore) FinishWeeklyDelivery(_ context.Context, id int64, status string, attempts int16, sentAt *time.Time, sendError string) error {
	for index := range s.deliveries {
		if s.deliveries[index].ID == id {
			s.deliveries[index].Status = status
			s.deliveries[index].Attempts = attempts
			s.deliveries[index].SentAt = sentAt
			s.deliveries[index].Error = sendError
			return nil
		}
	}
	return fmt.Errorf("delivery %d not found", id)
}

type sentMessage struct {
	userID int64
	text   string
}

type fakeSender struct {
	failures map[int64]int
	calls    map[int64]int
	messages []sentMessage
}

func (s *fakeSender) Send(_ context.Context, userID int64, text string) error {
	if s.calls == nil {
		s.calls = make(map[int64]int)
	}
	s.calls[userID]++
	if s.failures[userID] >= s.calls[userID] {
		return fmt.Errorf("MAX is unavailable")
	}
	s.messages = append(s.messages, sentMessage{userID: userID, text: text})
	return nil
}
