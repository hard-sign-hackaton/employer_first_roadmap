package employerapi

import (
	"context"
	"os"
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
	catalog, err := NewStore(tx).SaveCatalog(ctx, 0, request)
	if err != nil {
		t.Fatalf("save employer catalog: %v", err)
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
}
