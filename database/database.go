// Package database отвечает за подключение PostgreSQL и миграции GORM-моделей.
package database

import (
	"context"
	"fmt"
	"os"

	"efr_bot/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open открывает подключение к PostgreSQL и проверяет, что база отвечает.
func Open(ctx context.Context) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsnFromEnv()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get SQL database: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return db, nil
}

// AutoMigrate создаёт отсутствующие таблицы и добавляет безопасные изменения схемы.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Region{},
		&models.InterestTag{},
		&models.ExamSubject{},
		&models.Company{},
		&models.CareerDirection{},
		&models.CareerDirectionInterestTag{},
		&models.CompanyOpportunity{},
		&models.University{},
		&models.EducationProgram{},
		&models.CareerDirectionEducationProgram{},
		&models.ExamCombination{},
		&models.ExamCombinationItem{},
		&models.AdmissionScoreHistory{},
		&models.AdmissionCampaignRule{},
		&models.APIAccount{},
		&models.UserProfile{},
		&models.UserInterest{},
		&models.UserSubject{},
		&models.UserGoal{},
		&models.RoadmapTemplate{},
		&models.RoadmapTemplateStep{},
		&models.Roadmap{},
		&models.RoadmapStep{},
		&models.RoadmapAdmissionApplication{},
		&models.RoadmapEnrollmentChoice{},
		&models.RoadmapEmployerApplication{},
		&models.EmployerOpportunityAttempt{},
		&models.EmployerFeedback{},
		&models.RoadmapReminderDelivery{},
	); err != nil {
		return err
	}
	// До появления отдельных попыток feedback был связан внешним ключом с
	// единственной legacy-заявкой roadmap. Новая модель допускает несколько
	// попыток, поэтому эта старая связь больше не должна блокировать запись.
	return db.Exec("ALTER TABLE employer_feedbacks DROP CONSTRAINT IF EXISTS fk_roadmap_employer_applications_feedbacks").Error
}

func dsnFromEnv() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_PORT", "5432"),
		envOrDefault("DB_USER", "efr"),
		envOrDefault("DB_PASSWORD", "efr_local_password"),
		envOrDefault("DB_NAME", "employer_first_roadmap"),
		envOrDefault("DB_SSLMODE", "disable"),
	)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
