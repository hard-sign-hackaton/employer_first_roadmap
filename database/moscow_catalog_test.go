package database

import (
	"context"
	"os"
	"testing"
	"time"

	"efr_bot/models"
)

func TestSeedMoscowCatalogIsIdempotentAndDoesNotCreateDemoCatalog(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	// The efr_test database is disposable and test packages run serially.  Clear
	// catalog tables so this test proves that this seed itself adds no demo data.
	if err := db.Exec("TRUNCATE companies, universities, career_directions, education_programs, career_direction_education_programs, company_opportunities, roadmap_templates CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := SeedMoscowCatalog(db); err != nil {
		t.Fatal(err)
	}
	if err := SeedMoscowCatalog(db); err != nil {
		t.Fatal(err)
	}

	var companies []models.Company
	if err := db.Order("name").Find(&companies).Error; err != nil {
		t.Fatal(err)
	}
	if len(companies) != 7 {
		t.Fatalf("companies = %d, want 7", len(companies))
	}
	for _, company := range companies {
		if company.Name == "Т1" || company.Name == "ПАО «КАМАЗ»" {
			t.Fatalf("demo company %q leaked into Moscow catalog", company.Name)
		}
	}
	var universities []models.University
	if err := db.Preload("Region").Find(&universities).Error; err != nil {
		t.Fatal(err)
	}
	if len(universities) != 8 {
		t.Fatalf("universities = %d, want 8", len(universities))
	}
	for _, university := range universities {
		if university.Region.Name != "Москва" {
			t.Fatalf("university %q has region %q", university.Name, university.Region.Name)
		}
	}
	var opportunities []models.CompanyOpportunity
	if err := db.Find(&opportunities).Error; err != nil {
		t.Fatal(err)
	}
	if len(opportunities) < 7 {
		t.Fatalf("opportunities = %d, want at least 7", len(opportunities))
	}
	for _, opportunity := range opportunities {
		if opportunity.URL == "" || opportunity.SourceCheckedAt == nil || opportunity.WorkFormat == "" {
			t.Fatalf("opportunity %+v has incomplete source metadata", opportunity)
		}
		if opportunity.WorkFormat == "remote" && opportunity.RegionID != nil {
			t.Fatalf("remote opportunity must be nationwide, got region %d", *opportunity.RegionID)
		}
	}
	var links int64
	if err := db.WithContext(context.Background()).Model(&models.CareerDirectionEducationProgram{}).Count(&links).Error; err != nil {
		t.Fatal(err)
	}
	if links == 0 {
		t.Fatal("expected confirmed direction-program links")
	}
	var programs []models.EducationProgram
	if err := db.Preload("ExamCombinations.Items.ExamSubject").Preload("AdmissionScores").Find(&programs).Error; err != nil {
		t.Fatal(err)
	}
	if len(programs) != 23 {
		t.Fatalf("programs = %d, want 23", len(programs))
	}
	var hse models.EducationProgram
	if err := db.Where("code = ? AND name = ?", "01.03.02", "Компьютерные науки и анализ данных").Preload("ExamCombinations.Items.ExamSubject").First(&hse).Error; err != nil {
		t.Fatal(err)
	}
	if hse.SourceURL != "https://ba.hse.ru/minkrit" || hse.SourceCheckedAt == nil {
		t.Fatalf("HSE source metadata is not preserved: %+v", hse)
	}
	var informaticsMinimum *int16
	for _, combination := range hse.ExamCombinations {
		for _, item := range combination.Items {
			if item.ExamSubject.Name == "Информатика" {
				informaticsMinimum = item.MinScore
			}
		}
	}
	if informaticsMinimum == nil || *informaticsMinimum != 65 {
		t.Fatalf("HSE informatics minimum = %v, want 65", informaticsMinimum)
	}
}
