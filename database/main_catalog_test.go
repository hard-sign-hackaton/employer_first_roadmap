package database

import (
	"context"
	"os"
	"testing"
	"time"

	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/repositories"
	"efr_bot/services"
	"gorm.io/gorm"
)

func TestSeedMainCatalogIsIdempotentAndDoesNotCreateDemoCatalog(t *testing.T) {
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
	if err := db.Exec("TRUNCATE companies, universities, career_directions, education_programs, career_direction_education_programs, company_opportunities, roadmap_templates, admission_campaign_rules CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := SeedMainCatalog(db); err != nil {
		t.Fatal(err)
	}
	if err := SeedMainCatalog(db); err != nil {
		t.Fatal(err)
	}
	var campaignRule models.AdmissionCampaignRule
	if err := db.Order("admission_year DESC").First(&campaignRule).Error; err != nil {
		t.Fatalf("main catalog must seed admission campaign limits: %v", err)
	}
	if campaignRule.MaxUniversities != 5 || campaignRule.MaxProgramsPerUniversity != 5 {
		t.Fatalf("campaign limits = %+v, want 5 universities and 5 programs per university", campaignRule)
	}
	assertMainCatalogAdmissionPlanCanBeSaved(t, db)

	var companies []models.Company
	if err := db.Order("name").Find(&companies).Error; err != nil {
		t.Fatal(err)
	}
	if len(companies) != 10 {
		t.Fatalf("companies = %d, want 10", len(companies))
	}
	for _, company := range companies {
		if company.Name == "Т1" || company.Name == "ПАО «КАМАЗ»" {
			t.Fatalf("demo company %q leaked into main catalog", company.Name)
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
	if len(opportunities) != 19 {
		t.Fatalf("opportunities = %d, want 19", len(opportunities))
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
	if len(programs) != 26 {
		t.Fatalf("programs = %d, want 26", len(programs))
	}
	var infosec models.EducationProgram
	if err := db.Where("code = ? AND name = ?", "10.03.01", "Информационная безопасность").First(&infosec).Error; err != nil {
		t.Fatalf("expected verified MEPhI information-security programme: %v", err)
	}
	var kasperskyOpportunity models.CompanyOpportunity
	if err := db.Joins("JOIN companies ON companies.id = company_opportunities.company_id").
		Where("companies.name = ? AND company_opportunities.active_listing_url = ?", "Лаборатория Касперского", "https://careers.kaspersky.ru/vacancy/25720").
		First(&kasperskyOpportunity).Error; err != nil {
		t.Fatalf("expected Kaspersky's active Moscow information-security internship: %v", err)
	}
	if kasperskyOpportunity.RegionID == nil || kasperskyOpportunity.WorkFormat != "onsite" {
		t.Fatalf("Kaspersky opportunity must be Moscow onsite: %+v", kasperskyOpportunity)
	}
	var x5RemoteOpportunity models.CompanyOpportunity
	if err := db.Joins("JOIN companies ON companies.id = company_opportunities.company_id").
		Where("companies.name = ? AND company_opportunities.work_format = ?", "X5 Group", "remote").
		First(&x5RemoteOpportunity).Error; err != nil {
		t.Fatalf("expected verified remote X5 opportunity: %v", err)
	}
	if x5RemoteOpportunity.RegionID != nil {
		t.Fatalf("X5 remote opportunity must not be restricted to Moscow: %+v", x5RemoteOpportunity)
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

func assertMainCatalogAdmissionPlanCanBeSaved(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	var link models.CareerDirectionEducationProgram
	if err := db.First(&link).Error; err != nil {
		t.Fatalf("load main catalog direction-program link: %v", err)
	}
	var program models.EducationProgram
	if err := db.Preload("University").Preload("ExamCombinations.Items").First(&program, link.EducationProgramID).Error; err != nil {
		t.Fatalf("load main catalog programme: %v", err)
	}
	if len(program.ExamCombinations) == 0 || len(program.ExamCombinations[0].Items) == 0 {
		t.Fatal("main catalog programme must have an EGE combination")
	}

	profileStore := repositories.NewGormProfileRepository(db)
	careerStore := repositories.NewGormCareerRepository(db)
	educationStore := repositories.NewGormEducationRepository(db)
	roadmapStore := repositories.NewGormRoadmapRepository(db)
	profileService := services.NewProfileService(profileStore)
	trajectoryService := services.NewTrajectoryService(careerStore, educationStore, profileStore, roadmapStore)
	roadmapService := services.NewRoadmapService(roadmapStore, careerStore)
	admissionService := services.NewAdmissionService(educationStore, profileStore, roadmapStore, careerStore)

	userID := time.Now().UnixNano()
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 11, RegionID: program.University.RegionID, WillingToRelocate: true}); err != nil {
		t.Fatalf("save main catalog test profile: %v", err)
	}
	results := make([]dto.ExamResultInput, 0, len(program.ExamCombinations[0].Items))
	for _, item := range program.ExamCombinations[0].Items {
		results = append(results, dto.ExamResultInput{ExamSubjectID: item.ExamSubjectID, ActualScore: 100})
	}
	if _, err := profileService.SaveExamResults(ctx, userID, dto.SaveExamResultsRequest{Results: results}); err != nil {
		t.Fatalf("save main catalog test EGE results: %v", err)
	}
	goal, err := trajectoryService.ConfirmGoal(ctx, userID, dto.ConfirmGoalRequest{CareerDirectionID: link.CareerDirectionID, TargetAdmissionYear: 2026})
	if err != nil {
		t.Fatalf("create main catalog test goal: %v", err)
	}
	roadmap, err := roadmapService.CreateRoadmap(ctx, userID, dto.CreateRoadmapRequest{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("create main catalog test roadmap: %v", err)
	}
	applications, err := admissionService.SaveAdmissionPlan(ctx, userID, dto.GetAdmissionPlanRequest{RoadmapID: roadmap.ID}, dto.SaveAdmissionPlanRequest{Applications: []dto.AdmissionPlanItemInput{{EducationProgramID: program.ID}}})
	if err != nil || len(applications) != 1 {
		t.Fatalf("save admission plan from main catalogue: applications=%#v err=%v", applications, err)
	}
}
