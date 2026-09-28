package integration

import (
	"context"
	"strings"
	"testing"

	"efr_bot/dto"
	"efr_bot/models"
	"gorm.io/gorm"
)

func TestFullSurveyRanksCompaniesBySavedInterests(t *testing.T) {
	runScenario(t, "полный опрос: интересы → ранжирование компаний", testFullSurveyRanksCompaniesBySavedInterests)
}

func testFullSurveyRanksCompaniesBySavedInterests(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileService, trajectoryService, _ := scenarioServices(tx)
	ctx := context.Background()
	userID := scenarioUserID + 10

	regionID := regionIDByName(t, db, "Пермский край")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 10, RegionID: regionID}); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	programming := tagIDByName(t, db, "Программирование")
	mathematics := tagIDByName(t, db, "Математика")
	interests, err := profileService.SaveSurveyInterests(ctx, userID, dto.SaveSurveyInterestsRequest{Interests: []dto.InterestInput{
		{InterestTagID: programming, Weight: 2},
		{InterestTagID: mathematics, Weight: 1},
	}})
	if err != nil {
		t.Fatalf("save interests: %v", err)
	}
	if len(interests) != 2 {
		t.Fatalf("saved interests = %d, want 2", len(interests))
	}

	companies, err := trajectoryService.RecommendCompanies(ctx, userID, dto.RecommendCompaniesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("recommend companies: %v", err)
	}
	if len(companies) == 0 {
		t.Fatal("full survey must return companies for matching interests")
	}
	if companies[0].Name != "Т1" {
		t.Fatalf("top company = %q, want Т1 for programming and mathematics interests", companies[0].Name)
	}
	if len(companies[0].Reasons) == 0 {
		t.Fatal("recommended company must include an explanation")
	}
}

func TestEleventhGradeFiltersDirectionsBySelectedExamsAndScores(t *testing.T) {
	runScenario(t, "11 класс: фильтр направлений по ЕГЭ и баллам", testEleventhGradeFiltersDirectionsBySelectedExamsAndScores)
}

func TestFullSurveyFiltersCatalogByRegionAndReturnsDescriptions(t *testing.T) {
	runScenario(t, "полный опрос: региональный фильтр и описания карточек", testFullSurveyFiltersCatalogByRegionAndReturnsDescriptions)
}

func TestCompanyRecommendationDiagnosisExplainsRelocationOrMissingCatalogPath(t *testing.T) {
	runScenario(t, "полный опрос: диагностика отсутствия полного пути", testCompanyRecommendationDiagnosisExplainsRelocationOrMissingCatalogPath)
}

func testFullSurveyFiltersCatalogByRegionAndReturnsDescriptions(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileService, trajectoryService, _ := scenarioServices(tx)
	ctx := context.Background()
	userID := scenarioUserID + 12
	permID := regionIDByName(t, db, "Пермский край")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 10, RegionID: permID, WillingToRelocate: false}); err != nil {
		t.Fatalf("save local profile: %v", err)
	}

	companies, err := trajectoryService.RecommendCompanies(ctx, userID, dto.RecommendCompaniesRequest{Limit: 20})
	if err != nil {
		t.Fatalf("recommend local companies: %v", err)
	}
	t1 := findRecommendedCompany(t, companies, "Т1")
	if t1.Description == "" || strings.Contains(t1.Description, "Демонстрационная запись") {
		t.Fatalf("T1 description = %q, want friendly catalog description", t1.Description)
	}
	if t1.WebsiteURL != "https://t1.ru/" {
		t.Fatalf("T1 website URL = %q, want official URL", t1.WebsiteURL)
	}
	if containsRecommendedCompany(companies, "Группа компаний «МЕДСИ»") {
		t.Fatal("MEDSI must be hidden without relocation: it has no complete path in Perm")
	}

	directions, err := trajectoryService.GetCareerDirections(ctx, userID, dto.GetCareerDirectionsRequest{CompanyID: t1.ID})
	if err != nil {
		t.Fatalf("get local T1 directions: %v", err)
	}
	if len(directions) != 1 || directions[0].Name != "Разработчик программного обеспечения" {
		t.Fatalf("local T1 directions = %#v, want only software developer", directions)
	}
	if directions[0].Description == "" || strings.Contains(directions[0].Description, "Демонстрационное") {
		t.Fatalf("direction description = %q, want friendly description", directions[0].Description)
	}

	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 10, RegionID: permID, WillingToRelocate: true}); err != nil {
		t.Fatalf("allow relocation: %v", err)
	}
	companies, err = trajectoryService.RecommendCompanies(ctx, userID, dto.RecommendCompaniesRequest{Limit: 20})
	if err != nil {
		t.Fatalf("recommend all-region companies: %v", err)
	}
	if !containsRecommendedCompany(companies, "Группа компаний «МЕДСИ»") {
		t.Fatal("MEDSI must be available after allowing relocation")
	}
	directions, err = trajectoryService.GetCareerDirections(ctx, userID, dto.GetCareerDirectionsRequest{CompanyID: t1.ID})
	if err != nil {
		t.Fatalf("get all-region T1 directions: %v", err)
	}
	if len(directions) < 2 {
		t.Fatalf("all-region T1 directions = %#v, want remote directions too", directions)
	}
}

func testCompanyRecommendationDiagnosisExplainsRelocationOrMissingCatalogPath(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileService, trajectoryService, _ := scenarioServices(tx)
	ctx := context.Background()
	userID := scenarioUserID + 13
	spbID := regionIDByName(t, db, "Санкт-Петербург")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 9, RegionID: spbID, WillingToRelocate: false}); err != nil {
		t.Fatalf("save local profile: %v", err)
	}
	diagnosis, err := trajectoryService.DiagnoseCompanyRecommendations(ctx, userID)
	if err != nil {
		t.Fatalf("diagnose local recommendations: %v", err)
	}
	if diagnosis.Issue != dto.CompanyRecommendationIssueRelocationRequired {
		t.Fatalf("local diagnosis = %q, want relocation_required", diagnosis.Issue)
	}

	if err := tx.Exec("DELETE FROM company_opportunities").Error; err != nil {
		t.Fatalf("remove opportunities for catalog-path scenario: %v", err)
	}
	diagnosis, err = trajectoryService.DiagnoseCompanyRecommendations(ctx, userID)
	if err != nil {
		t.Fatalf("diagnose missing catalog path: %v", err)
	}
	if diagnosis.Issue != dto.CompanyRecommendationIssueNoCatalogPath {
		t.Fatalf("catalog diagnosis = %q, want no_catalog_path", diagnosis.Issue)
	}
}

func containsRecommendedCompany(companies []dto.RecommendedCompanyResponse, name string) bool {
	for _, company := range companies {
		if company.Name == name {
			return true
		}
	}
	return false
}

func findRecommendedCompany(t *testing.T, companies []dto.RecommendedCompanyResponse, name string) dto.RecommendedCompanyResponse {
	t.Helper()
	for _, company := range companies {
		if company.Name == name {
			return company
		}
	}
	t.Fatalf("recommended company %q was not returned", name)
	return dto.RecommendedCompanyResponse{}
}

func testEleventhGradeFiltersDirectionsBySelectedExamsAndScores(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileService, trajectoryService, _ := scenarioServices(tx)
	ctx := context.Background()
	userID := scenarioUserID + 11
	regionID := regionIDByName(t, db, "Пермский край")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 11, RegionID: regionID}); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	selected := selectedExamInputs(t, db, 100)
	if _, err := profileService.SaveUserSubjects(ctx, userID, dto.SaveUserSubjectsRequest{Subjects: selected}); err != nil {
		t.Fatalf("save selected EGE: %v", err)
	}
	company := companyByName(t, trajectoryService, ctx, "Т1")
	directions, err := trajectoryService.GetCareerDirections(ctx, userID, dto.GetCareerDirectionsRequest{CompanyID: company.ID})
	if err != nil {
		t.Fatalf("get filtered directions: %v", err)
	}
	if len(directions) != 1 || directions[0].Name != "Разработчик программного обеспечения" {
		t.Fatalf("filtered directions = %#v, want only software developer", directions)
	}

	companies, err := trajectoryService.RecommendCompanies(ctx, userID, dto.RecommendCompaniesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("recommend filtered companies: %v", err)
	}
	for _, company := range companies {
		if company.Name == "ПАО «КАМАЗ»" {
			t.Fatal("KAMAZ must be excluded: selected EGE set has no physics")
		}
	}

	lowScoreInputs := selectedExamInputs(t, db, 0)
	if _, err := profileService.SaveUserSubjects(ctx, userID, dto.SaveUserSubjectsRequest{Subjects: lowScoreInputs}); err != nil {
		t.Fatalf("save low expected scores: %v", err)
	}
	companies, err = trajectoryService.RecommendCompanies(ctx, userID, dto.RecommendCompaniesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("recommend companies for low scores: %v", err)
	}
	if len(companies) != 0 {
		t.Fatalf("companies = %d, want none when every expected score is below minimum", len(companies))
	}
}

func selectedExamInputs(t *testing.T, db *gorm.DB, score int16) []dto.UserSubjectInput {
	t.Helper()
	names := []string{"Русский язык", "Математика (профильная)", "Информатика"}
	inputs := make([]dto.UserSubjectInput, 0, len(names))
	for _, name := range names {
		var subject models.ExamSubject
		if err := db.First(&subject, "name = ?", name).Error; err != nil {
			t.Fatalf("find EGE subject %q: %v", name, err)
		}
		value := score
		inputs = append(inputs, dto.UserSubjectInput{ExamSubjectID: subject.ID, Status: models.SubjectStatusSelected, ExpectedScore: &value})
	}
	return inputs
}

func examResults(t *testing.T, db *gorm.DB, score int16, names ...string) []dto.ExamResultInput {
	t.Helper()
	results := make([]dto.ExamResultInput, 0, len(names))
	for _, name := range names {
		var subject models.ExamSubject
		if err := db.First(&subject, "name = ?", name).Error; err != nil {
			t.Fatalf("find EGE subject %q: %v", name, err)
		}
		results = append(results, dto.ExamResultInput{ExamSubjectID: subject.ID, ActualScore: score})
	}
	return results
}

func tagIDByName(t *testing.T, db *gorm.DB, name string) int64 {
	t.Helper()
	var tag models.InterestTag
	if err := db.First(&tag, "name = ?", name).Error; err != nil {
		t.Fatalf("find interest tag %q: %v", name, err)
	}
	return tag.ID
}
