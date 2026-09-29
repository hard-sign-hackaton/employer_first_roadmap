package database

import (
	"context"
	"os"
	"testing"
	"time"

	"efr_bot/models"
)

// Запускается только для явной проверки PostgreSQL, чтобы обычные go test не требовали БД.
func TestAutoMigratePostgres(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := Open(ctx)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	defer sqlDB.Close()

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	for _, model := range []any{
		&models.Region{},
		&models.UserProfile{},
		&models.Company{},
		&models.CareerDirection{},
		&models.University{},
		&models.EducationProgram{},
		&models.Roadmap{},
		&models.RoadmapStep{},
		&models.RoadmapAdmissionApplication{},
		&models.RoadmapReminderDelivery{},
	} {
		if !db.Migrator().HasTable(model) {
			t.Errorf("table for %T was not created", model)
		}
	}
}

func TestSeedReferenceDataPostgres(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := Open(ctx)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

	var companiesBefore int64
	if err := tx.Model(&models.Company{}).Count(&companiesBefore).Error; err != nil {
		t.Fatalf("count companies before reference seed: %v", err)
	}
	if err := SeedReferenceData(tx); err != nil {
		t.Fatalf("seed reference data: %v", err)
	}
	for model, minimum := range map[any]int64{
		&models.Region{}:      6,
		&models.ExamSubject{}: 12,
		&models.InterestTag{}: 17,
	} {
		var count int64
		if err := tx.Model(model).Count(&count).Error; err != nil {
			t.Fatalf("count %T: %v", model, err)
		}
		if count < minimum {
			t.Errorf("%T count = %d, want at least %d", model, count, minimum)
		}
	}
	var companiesAfter int64
	if err := tx.Model(&models.Company{}).Count(&companiesAfter).Error; err != nil {
		t.Fatalf("count companies after reference seed: %v", err)
	}
	if companiesAfter != companiesBefore {
		t.Fatalf("reference seed changed companies count from %d to %d", companiesBefore, companiesAfter)
	}
}

func TestSeedDemoDataPostgres(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := Open(ctx)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := SeedDemoData(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := SeedDemoData(db); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	for model, expected := range map[any]int64{
		&models.Region{}:           6,
		&models.ExamSubject{}:      12,
		&models.InterestTag{}:      17,
		&models.Company{}:          7,
		&models.CareerDirection{}:  23,
		&models.University{}:       10,
		&models.EducationProgram{}: 18,
		&models.RoadmapTemplate{}:  23,
	} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil {
			t.Fatalf("count %T: %v", model, err)
		}
		if count != expected {
			t.Errorf("%T count = %d, want %d", model, count, expected)
		}
	}
}
