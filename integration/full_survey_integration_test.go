package integration

import (
	"context"
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

func tagIDByName(t *testing.T, db *gorm.DB, name string) int64 {
	t.Helper()
	var tag models.InterestTag
	if err := db.First(&tag, "name = ?", name).Error; err != nil {
		t.Fatalf("find interest tag %q: %v", name, err)
	}
	return tag.ID
}
