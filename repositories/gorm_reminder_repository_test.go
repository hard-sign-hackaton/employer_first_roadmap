package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"efr_bot/database"
	"efr_bot/models"
)

func TestReminderTargetsUseFirstUnfinishedStepOnly(t *testing.T) {
	roadmaps := []models.Roadmap{
		{
			ID:       10,
			UserGoal: models.UserGoal{UserProfileID: 100},
			Steps: []models.RoadmapStep{
				{ID: 1, OrderNo: 1, Title: "Завершённый", Status: models.RoadmapStepStatusCompleted},
				{ID: 2, OrderNo: 2, Title: "Текущий", Status: models.RoadmapStepStatusActive},
				{ID: 3, OrderNo: 3, Title: "Следующий", Status: models.RoadmapStepStatusPending},
			},
		},
		{ID: 20, UserGoal: models.UserGoal{UserProfileID: 200}, Steps: []models.RoadmapStep{{ID: 4, Title: "Всё сделано", Status: models.RoadmapStepStatusCompleted}}},
	}

	targets := reminderTargets(roadmaps)
	if len(targets) != 1 {
		t.Fatalf("targets = %#v, want exactly one active roadmap", targets)
	}
	if target := targets[0]; target.UserID != 100 || target.RoadmapID != 10 || target.StepID != 2 || target.StepTitle != "Текущий" {
		t.Fatalf("target = %#v", target)
	}
}

func TestGormReminderRepositoryListsOnlyActiveRoadmaps(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := database.Open(ctx)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

	region := models.Region{Name: "Reminder test region " + time.Now().UTC().Format("150405.000000")}
	if err := tx.Create(&region).Error; err != nil {
		t.Fatalf("create region: %v", err)
	}
	profile := models.UserProfile{ID: 909001, Grade: 10, RegionID: region.ID}
	if err := tx.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	company := models.Company{Name: "Reminder test company " + region.Name, IsActive: true}
	if err := tx.Create(&company).Error; err != nil {
		t.Fatalf("create company: %v", err)
	}
	direction := models.CareerDirection{CompanyID: company.ID, Name: "Reminder test direction " + region.Name, IsActive: true}
	if err := tx.Create(&direction).Error; err != nil {
		t.Fatalf("create direction: %v", err)
	}
	template := models.RoadmapTemplate{CareerDirectionID: direction.ID, Name: "Reminder test template", Version: 1, IsActive: true}
	if err := tx.Create(&template).Error; err != nil {
		t.Fatalf("create template: %v", err)
	}
	goal := models.UserGoal{UserProfileID: profile.ID, CareerDirectionID: direction.ID, TargetAdmissionYear: 2027, Status: models.GoalStatusActive}
	if err := tx.Create(&goal).Error; err != nil {
		t.Fatalf("create goal: %v", err)
	}
	active := models.Roadmap{UserGoalID: goal.ID, RoadmapTemplateID: template.ID, Status: models.RoadmapStatusActive}
	archived := models.Roadmap{UserGoalID: goal.ID, RoadmapTemplateID: template.ID, Status: models.RoadmapStatusArchived}
	if err := tx.Create(&active).Error; err != nil {
		t.Fatalf("create active roadmap: %v", err)
	}
	if err := tx.Create(&archived).Error; err != nil {
		t.Fatalf("create archived roadmap: %v", err)
	}
	steps := []models.RoadmapStep{
		{RoadmapID: active.ID, OrderNo: 1, StepType: "test", Title: "Уже готово", Status: models.RoadmapStepStatusCompleted},
		{RoadmapID: active.ID, OrderNo: 2, StepType: "test", Title: "Текущий шаг", Status: models.RoadmapStepStatusActive},
		{RoadmapID: archived.ID, OrderNo: 1, StepType: "test", Title: "Архивный шаг", Status: models.RoadmapStepStatusActive},
	}
	if err := tx.Create(&steps).Error; err != nil {
		t.Fatalf("create steps: %v", err)
	}

	targets, err := NewGormReminderRepository(tx).ListActiveTargets(ctx)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}
	var found bool
	for _, target := range targets {
		if target.RoadmapID == active.ID {
			found = true
			if target.UserID != profile.ID || target.StepTitle != "Текущий шаг" {
				t.Fatalf("active target = %#v", target)
			}
		}
		if target.RoadmapID == archived.ID {
			t.Fatalf("archived roadmap must not be a target: %#v", target)
		}
	}
	if !found {
		t.Fatal("active roadmap target not found")
	}
}
