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
	} {
		if !db.Migrator().HasTable(model) {
			t.Errorf("table for %T was not created", model)
		}
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
		&models.Region{}:           3,
		&models.ExamSubject{}:      8,
		&models.InterestTag{}:      8,
		&models.Company{}:          3,
		&models.CareerDirection{}:  5,
		&models.University{}:       4,
		&models.EducationProgram{}: 5,
		&models.RoadmapTemplate{}:  5,
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
