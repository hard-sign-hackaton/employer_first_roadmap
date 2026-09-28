package employerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"efr_bot/database"
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/repositories"
	"efr_bot/services"
)

// This proves the important boundary: data written through the employer store
// is immediately usable by the existing user-side bot services.
func TestEmployerCatalogFeedsUserRoadmap(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

	request := validCatalogRequest()
	request.Company.Name = "Integration Employer API"
	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal employer catalog: %v", err)
	}
	handler := NewHandler(NewStore(tx), "integration-secret")
	httpRequest := httptest.NewRequest(http.MethodPost, "/api/v1/employer/catalog", bytes.NewReader(requestBody)).WithContext(ctx)
	httpRequest.Header.Set("Authorization", "Bearer integration-secret")
	httpResponse := httptest.NewRecorder()
	handler.ServeHTTP(httpResponse, httpRequest)
	if httpResponse.Code != http.StatusCreated {
		t.Fatalf("create employer catalog HTTP status = %d, body = %s", httpResponse.Code, httpResponse.Body.String())
	}
	var catalog CatalogResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode employer catalog response: %v", err)
	}
	if len(catalog.Directions) != 1 || len(catalog.Directions[0].EducationPrograms) != 1 || len(catalog.Directions[0].Opportunities) != 1 {
		t.Fatalf("incomplete saved catalog: %#v", catalog)
	}

	profiles := repositories.NewGormProfileRepository(tx)
	careers := repositories.NewGormCareerRepository(tx)
	education := repositories.NewGormEducationRepository(tx)
	roadmaps := repositories.NewGormRoadmapRepository(tx)
	profileService := services.NewProfileService(profiles)
	trajectoryService := services.NewTrajectoryService(careers, education, profiles, roadmaps)
	roadmapService := services.NewRoadmapService(roadmaps, careers)
	const userID int64 = 9_800_001
	regionID := catalog.Directions[0].EducationPrograms[0].University.RegionID
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 10, RegionID: regionID}); err != nil {
		t.Fatalf("save user profile: %v", err)
	}
	directions, err := trajectoryService.GetCareerDirections(ctx, userID, dto.GetCareerDirectionsRequest{CompanyID: catalog.Company.ID})
	if err != nil || len(directions) != 1 {
		t.Fatalf("bot directions = %#v, error = %v", directions, err)
	}
	examSets, err := trajectoryService.GetRecommendedExamSets(ctx, dto.GetRecommendedExamSetsRequest{CareerDirectionID: directions[0].ID})
	if err != nil || len(examSets) == 0 {
		t.Fatalf("bot exam sets = %#v, error = %v", examSets, err)
	}
	subjects := make([]dto.UserSubjectInput, 0, len(examSets[0].ExamSubjectIDs))
	for _, subjectID := range examSets[0].ExamSubjectIDs {
		subjects = append(subjects, dto.UserSubjectInput{ExamSubjectID: subjectID, Status: models.SubjectStatusPlanned})
	}
	if _, err := profileService.SaveUserSubjects(ctx, userID, dto.SaveUserSubjectsRequest{Subjects: subjects}); err != nil {
		t.Fatalf("save user subjects: %v", err)
	}
	goal, err := trajectoryService.ConfirmGoal(ctx, userID, dto.ConfirmGoalRequest{CareerDirectionID: directions[0].ID, TargetAdmissionYear: 2027})
	if err != nil {
		t.Fatalf("confirm bot goal: %v", err)
	}
	roadmap, err := roadmapService.CreateRoadmap(ctx, userID, dto.CreateRoadmapRequest{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("create bot roadmap: %v", err)
	}
	if len(roadmap.Steps) != len(canonicalRoadmapSteps) {
		t.Fatalf("roadmap steps = %d, want %d", len(roadmap.Steps), len(canonicalRoadmapSteps))
	}

	application := models.RoadmapEmployerApplication{
		RoadmapID: roadmap.ID, CompanyOpportunityID: catalog.Directions[0].Opportunities[0].ID,
		Status: "submitted", SubmittedAt: time.Now().UTC(),
	}
	if err := tx.Create(&application).Error; err != nil {
		t.Fatalf("create submitted application: %v", err)
	}
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/employer/companies/"+fmt.Sprint(catalog.Company.ID)+"/applications", nil).WithContext(ctx)
	listRequest.Header.Set("Authorization", "Bearer integration-secret")
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "submitted") {
		t.Fatalf("list applications status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}
	feedbackBody := []byte(`{"status":"interview","message":"Приглашаем на интервью","contact":"hr@example.com"}`)
	feedbackRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/employer/applications/"+fmt.Sprint(roadmap.ID), bytes.NewReader(feedbackBody)).WithContext(ctx)
	feedbackRequest.Header.Set("Authorization", "Bearer integration-secret")
	feedbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(feedbackResponse, feedbackRequest)
	if feedbackResponse.Code != http.StatusOK {
		t.Fatalf("save feedback status = %d, body = %s", feedbackResponse.Code, feedbackResponse.Body.String())
	}
	userFeedback, err := roadmapService.GetEmployerFeedback(ctx, userID)
	if err != nil {
		t.Fatalf("get feedback in bot service: %v", err)
	}
	if userFeedback.Status != "interview" || userFeedback.Message != "Приглашаем на интервью" || userFeedback.Contact != "hr@example.com" {
		t.Fatalf("unexpected bot feedback: %#v", userFeedback)
	}
}
