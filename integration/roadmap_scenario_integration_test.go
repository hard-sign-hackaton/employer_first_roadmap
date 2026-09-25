// Package integration проверяет бизнес-сценарии через реальные сервисы,
// репозитории GORM и изолированную PostgreSQL, но без подключения к MAX.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"efr_bot/database"
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/repositories"
	"efr_bot/services"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const scenarioUserID int64 = 9_001_001

type scenarioTestStats struct {
	total  int
	passed int
	failed int
}

var stats scenarioTestStats

// TestMain печатает компактную сводку после всех интеграционных сценариев.
func TestMain(m *testing.M) {
	exitCode := m.Run()
	fmt.Printf("\nИнтеграционные сценарии: всего %d, пройдено %d, упало %d\n", stats.total, stats.passed, stats.failed)
	os.Exit(exitCode)
}

func runScenario(t *testing.T, name string, test func(t *testing.T)) {
	t.Helper()
	stats.total++
	defer func() {
		if t.Failed() {
			stats.failed++
			return
		}
		stats.passed++
	}()
	t.Run(name, test)
}

// TestSmallSurveyScenarioCreatesRoadmap повторяет реализованный путь 9–10 класса:
// работодатель → профиль → направление → набор ЕГЭ → цель → roadmap.
func TestSmallSurveyScenarioCreatesRoadmap(t *testing.T) {
	runScenario(t, "10 класс: компания → ЕГЭ → цель → roadmap", testSmallSurveyScenarioCreatesRoadmap)
}

func testSmallSurveyScenarioCreatesRoadmap(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin scenario transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

	profileStore := repositories.NewGormProfileRepository(tx)
	careerStore := repositories.NewGormCareerRepository(tx)
	educationStore := repositories.NewGormEducationRepository(tx)
	roadmapStore := repositories.NewGormRoadmapRepository(tx)

	profileService := services.NewProfileService(profileStore)
	trajectoryService := services.NewTrajectoryService(careerStore, educationStore, profileStore, roadmapStore)
	roadmapService := services.NewRoadmapService(roadmapStore, careerStore)
	ctx := context.Background()

	company := companyByName(t, trajectoryService, ctx, "Т1")
	regionID := regionIDByName(t, db, "Пермский край")

	if _, err := profileService.SaveProfile(ctx, scenarioUserID, dto.UpsertProfileRequest{
		Grade:             10,
		RegionID:          regionID,
		WillingToRelocate: false,
	}); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	selectedCompany, err := trajectoryService.SelectCompany(ctx, scenarioUserID, dto.SelectCompanyRequest{CompanyID: company.ID})
	if err != nil {
		t.Fatalf("select company: %v", err)
	}
	if selectedCompany.Name != "Т1" {
		t.Fatalf("selected company = %q, want Т1", selectedCompany.Name)
	}

	directions, err := trajectoryService.GetCareerDirections(ctx, scenarioUserID, dto.GetCareerDirectionsRequest{CompanyID: company.ID})
	if err != nil {
		t.Fatalf("get career directions: %v", err)
	}
	direction := directionByName(t, directions, "Разработчик программного обеспечения")

	examSets, err := trajectoryService.GetRecommendedExamSets(ctx, dto.GetRecommendedExamSetsRequest{CareerDirectionID: direction.ID})
	if err != nil {
		t.Fatalf("get recommended exam sets: %v", err)
	}
	if len(examSets) == 0 || len(examSets[0].ExamSubjectIDs) == 0 {
		t.Fatal("expected at least one non-empty EGE set")
	}
	if examSets[0].SourceYear != 2026 {
		t.Fatalf("source year = %d, want latest published demo rules for 2026", examSets[0].SourceYear)
	}

	plannedSubjects := make([]dto.UserSubjectInput, 0, len(examSets[0].ExamSubjectIDs))
	for _, subjectID := range examSets[0].ExamSubjectIDs {
		plannedSubjects = append(plannedSubjects, dto.UserSubjectInput{
			ExamSubjectID: subjectID,
			Status:        models.SubjectStatusPlanned,
		})
	}
	savedSubjects, err := profileService.SaveUserSubjects(ctx, scenarioUserID, dto.SaveUserSubjectsRequest{Subjects: plannedSubjects})
	if err != nil {
		t.Fatalf("save planned EGE subjects: %v", err)
	}
	if len(savedSubjects) != len(plannedSubjects) {
		t.Fatalf("saved subjects = %d, want %d", len(savedSubjects), len(plannedSubjects))
	}

	targetYear := expectedTargetAdmissionYear(10, time.Now())
	goal, err := trajectoryService.ConfirmGoal(ctx, scenarioUserID, dto.ConfirmGoalRequest{
		CareerDirectionID:   direction.ID,
		TargetAdmissionYear: targetYear,
	})
	if err != nil {
		t.Fatalf("confirm goal: %v", err)
	}
	if goal.Company.Name != "Т1" || goal.CareerDirection.Name != direction.Name {
		t.Fatalf("unexpected goal: company=%q direction=%q", goal.Company.Name, goal.CareerDirection.Name)
	}
	if goal.TargetAdmissionYear != targetYear {
		t.Fatalf("target admission year = %d, want %d", goal.TargetAdmissionYear, targetYear)
	}
	if len(goal.Subjects) != len(plannedSubjects) {
		t.Fatalf("goal contains %d subjects, want %d", len(goal.Subjects), len(plannedSubjects))
	}

	roadmap, err := roadmapService.CreateRoadmap(ctx, scenarioUserID, dto.CreateRoadmapRequest{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("create roadmap: %v", err)
	}
	if roadmap.Status != models.RoadmapStatusActive {
		t.Fatalf("roadmap status = %q, want %q", roadmap.Status, models.RoadmapStatusActive)
	}
	if len(roadmap.Steps) == 0 || roadmap.NextAction == nil {
		t.Fatal("roadmap must contain steps and a next action")
	}
	resumedRoadmap, err := roadmapService.GetActiveRoadmap(ctx, scenarioUserID)
	if err != nil {
		t.Fatalf("resume active roadmap: %v", err)
	}
	if resumedRoadmap.ID != roadmap.ID || resumedRoadmap.NextAction == nil || resumedRoadmap.NextAction.StepType != models.RoadmapStepTypeChooseOrConfirmExams {
		t.Fatalf("unexpected resumed roadmap: %#v", resumedRoadmap)
	}
	if err := roadmapService.RestartActiveRoadmap(ctx, scenarioUserID); err != nil {
		t.Fatalf("restart active roadmap: %v", err)
	}
	if _, err := roadmapService.GetActiveRoadmap(ctx, scenarioUserID); err == nil {
		t.Fatal("no active roadmap must remain after restart")
	}
	if _, err := roadmapService.GetRoadmap(ctx, scenarioUserID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID}); err == nil {
		t.Fatal("roadmap must be deleted after restart")
	}
	for name, model := range map[string]any{
		"profile":  &models.UserProfile{},
		"subjects": &models.UserSubject{},
		"goals":    &models.UserGoal{},
		"roadmaps": &models.Roadmap{},
	} {
		var count int64
		query := tx.Model(model)
		if name == "profile" {
			query = query.Where("id = ?", scenarioUserID)
		} else if name == "roadmaps" {
			query = query.Joins("JOIN user_goals ON user_goals.id = roadmaps.user_goal_id").Where("user_goals.user_profile_id = ?", scenarioUserID)
		} else {
			query = query.Where("user_profile_id = ?", scenarioUserID)
		}
		if err := query.Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%s must be deleted after restart: count=%d err=%v", name, count, err)
		}
	}
}

// TestSmallSurveyScenarioForNinthGrade проверяет отдельную развилку 9 класса.
// Она должна сформировать тот же общий roadmap, но с более поздним годом поступления.
func TestSmallSurveyScenarioForNinthGrade(t *testing.T) {
	runScenario(t, "9 класс: компания → ЕГЭ → цель → roadmap", testSmallSurveyScenarioForNinthGrade)
}

// TestAdmissionScenarioUsesLatestPublishedRules проверяет путь после ЕГЭ для
// будущего года поступления: подбор использует последние опубликованные правила,
// а план подачи — последние опубликованные лимиты кампании.
func TestAdmissionScenarioUsesLatestPublishedRules(t *testing.T) {
	runScenario(t, "будущее поступление: последние правила → вуз → зачисление → работодатель", testAdmissionScenarioUsesLatestPublishedRules)
}

func TestEnrollmentLifecycleRestartsOrReachesEmployer(t *testing.T) {
	runScenario(t, "приёмная кампания: не поступил → новый опрос; поступил → курс → заявка работодателю", testEnrollmentLifecycleRestartsOrReachesEmployer)
}

func testEnrollmentLifecycleRestartsOrReachesEmployer(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileStore := repositories.NewGormProfileRepository(tx)
	careerStore := repositories.NewGormCareerRepository(tx)
	educationStore := repositories.NewGormEducationRepository(tx)
	roadmapStore := repositories.NewGormRoadmapRepository(tx)
	profileService := services.NewProfileService(profileStore)
	trajectoryService := services.NewTrajectoryService(careerStore, educationStore, profileStore, roadmapStore)
	roadmapService := services.NewRoadmapService(roadmapStore, careerStore)
	admissionService := services.NewAdmissionService(educationStore, profileStore, roadmapStore, careerStore)
	ctx := context.Background()
	userID := scenarioUserID + 30
	regionID := regionIDByName(t, db, "Пермский край")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 11, RegionID: regionID}); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	company := companyByName(t, trajectoryService, ctx, "Т1")
	if _, err := trajectoryService.SelectCompany(ctx, userID, dto.SelectCompanyRequest{CompanyID: company.ID}); err != nil {
		t.Fatalf("select company: %v", err)
	}
	direction := directionByName(t, mustDirections(t, trajectoryService, ctx, userID, company.ID), "Разработчик программного обеспечения")
	if _, err := profileService.SaveUserSubjects(ctx, userID, dto.SaveUserSubjectsRequest{Subjects: selectedExamInputs(t, db, 100)}); err != nil {
		t.Fatalf("save EGE: %v", err)
	}
	goal, err := trajectoryService.ConfirmGoal(ctx, userID, dto.ConfirmGoalRequest{CareerDirectionID: direction.ID, TargetAdmissionYear: 2026})
	if err != nil {
		t.Fatalf("confirm goal: %v", err)
	}
	roadmap, err := roadmapService.CreateRoadmap(ctx, userID, dto.CreateRoadmapRequest{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("create roadmap: %v", err)
	}
	if _, err := admissionService.SaveEnrollmentChoice(ctx, userID, dto.GetAdmissionPlanRequest{RoadmapID: roadmap.ID}, dto.SaveEnrollmentChoiceRequest{Status: models.EnrollmentStatusNotEnrolled}); err != nil {
		t.Fatalf("save no-enrollment result: %v", err)
	}
	if _, err := roadmapService.GetActiveRoadmap(ctx, userID); err == nil {
		t.Fatal("roadmap must be archived after no-enrollment result")
	}
}

func testAdmissionScenarioUsesLatestPublishedRules(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileStore := repositories.NewGormProfileRepository(tx)
	careerStore := repositories.NewGormCareerRepository(tx)
	educationStore := repositories.NewGormEducationRepository(tx)
	roadmapStore := repositories.NewGormRoadmapRepository(tx)
	profileService := services.NewProfileService(profileStore)
	trajectoryService := services.NewTrajectoryService(careerStore, educationStore, profileStore, roadmapStore)
	roadmapService := services.NewRoadmapService(roadmapStore, careerStore)
	admissionService := services.NewAdmissionService(educationStore, profileStore, roadmapStore, careerStore)
	ctx := context.Background()
	userID := scenarioUserID + 3
	const targetAdmissionYear int16 = 2028

	regionID := regionIDByName(t, db, "Пермский край")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 10, RegionID: regionID, WillingToRelocate: false}); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	company := companyByName(t, trajectoryService, ctx, "Т1")
	if _, err := trajectoryService.SelectCompany(ctx, userID, dto.SelectCompanyRequest{CompanyID: company.ID}); err != nil {
		t.Fatalf("select company: %v", err)
	}
	direction := directionByName(t, mustDirections(t, trajectoryService, ctx, userID, company.ID), "Разработчик программного обеспечения")
	resultNames := []string{"Русский язык", "Математика (профильная)", "Информатика"}
	plannedSubjects := selectedExamInputs(t, db, 0)
	for index := range plannedSubjects {
		plannedSubjects[index].Status = models.SubjectStatusPlanned
		plannedSubjects[index].ExpectedScore = nil
	}
	if _, err := profileService.SaveUserSubjects(ctx, userID, dto.SaveUserSubjectsRequest{Subjects: plannedSubjects}); err != nil {
		t.Fatalf("save planned EGE subjects: %v", err)
	}
	goal, err := trajectoryService.ConfirmGoal(ctx, userID, dto.ConfirmGoalRequest{CareerDirectionID: direction.ID, TargetAdmissionYear: targetAdmissionYear})
	if err != nil {
		t.Fatalf("confirm goal: %v", err)
	}
	roadmap, err := roadmapService.CreateRoadmap(ctx, userID, dto.CreateRoadmapRequest{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("create roadmap: %v", err)
	}

	if _, err := profileService.SaveExamResults(ctx, userID, dto.SaveExamResultsRequest{Results: examResults(t, db, 100, "Русский язык", "История", "Литература")}); err != nil {
		t.Fatalf("save incompatible EGE results: %v", err)
	}
	assertEducationDiagnosis(t, admissionService, ctx, userID, roadmap.ID, "exam_subjects_mismatch")
	if _, err := profileService.SaveExamResults(ctx, userID, dto.SaveExamResultsRequest{Results: examResults(t, db, 0, resultNames...)}); err != nil {
		t.Fatalf("save low EGE results: %v", err)
	}
	assertEducationDiagnosis(t, admissionService, ctx, userID, roadmap.ID, "minimum_scores_not_met")
	moscowID := regionIDByName(t, db, "Москва")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 10, RegionID: moscowID, WillingToRelocate: false}); err != nil {
		t.Fatalf("save Moscow profile: %v", err)
	}
	if _, err := profileService.SaveExamResults(ctx, userID, dto.SaveExamResultsRequest{Results: examResults(t, db, 100, resultNames...)}); err != nil {
		t.Fatalf("save high EGE results: %v", err)
	}
	assertEducationDiagnosis(t, admissionService, ctx, userID, roadmap.ID, "region_restriction")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 10, RegionID: regionID, WillingToRelocate: false}); err != nil {
		t.Fatalf("restore Perm profile: %v", err)
	}

	options, err := admissionService.FindEducationOptions(ctx, userID, dto.FindEducationOptionsRequest{RoadmapID: roadmap.ID})
	if err != nil || len(options) == 0 {
		t.Fatalf("find education options: %v; options=%d", err, len(options))
	}
	if len(options) < 2 {
		t.Fatalf("education options = %d, want at least two programs for admission-plan test", len(options))
	}
	option := options[0]
	if option.AdmissionYear != targetAdmissionYear || option.RulesSourceYear != 2026 {
		t.Fatalf("education option years = target %d / rules %d, want %d / 2026", option.AdmissionYear, option.RulesSourceYear, targetAdmissionYear)
	}
	if option.PassingScoreSourceYear == nil || *option.PassingScoreSourceYear != 2025 {
		t.Fatalf("passing-score source year = %v, want 2025", option.PassingScoreSourceYear)
	}

	limits, err := admissionService.GetAdmissionPlanLimits(ctx, userID, dto.GetAdmissionPlanRequest{RoadmapID: roadmap.ID})
	if err != nil {
		t.Fatalf("get admission plan limits: %v", err)
	}
	if limits.AdmissionYear != targetAdmissionYear || limits.RulesSourceYear != 2026 || limits.MaxUniversities != 5 || limits.MaxProgramsPerUniversity != 5 {
		t.Fatalf("unexpected plan limits: %#v", limits)
	}
	applications, err := admissionService.SaveAdmissionPlan(ctx, userID, dto.GetAdmissionPlanRequest{RoadmapID: roadmap.ID}, dto.SaveAdmissionPlanRequest{Applications: []dto.AdmissionPlanItemInput{
		{EducationProgramID: option.EducationProgramID},
		{EducationProgramID: options[1].EducationProgramID},
	}})
	if err != nil || len(applications) != 2 {
		t.Fatalf("save admission plan: %v; applications=%d", err, len(applications))
	}
	savedPlan, err := admissionService.GetAdmissionPlan(ctx, userID, dto.GetAdmissionPlanRequest{RoadmapID: roadmap.ID})
	if err != nil || len(savedPlan) != 2 {
		t.Fatalf("get saved admission plan: %v; applications=%d", err, len(savedPlan))
	}
	if _, err := roadmapService.GetEmployerOpportunity(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID}); err == nil {
		t.Fatal("employer opportunity must not be available before final enrollment choice")
	}
	applicationID := applications[1].ID
	if _, err := admissionService.SaveEnrollmentChoice(ctx, userID, dto.GetAdmissionPlanRequest{RoadmapID: roadmap.ID}, dto.SaveEnrollmentChoiceRequest{Status: models.EnrollmentStatusChosen, AdmissionApplicationID: &applicationID}); err != nil {
		t.Fatalf("save enrollment choice: %v", err)
	}
	roadmapAfterEnrollment, err := roadmapService.GetRoadmap(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
	if err != nil || roadmapAfterEnrollment.EnrollmentChoice == nil || roadmapAfterEnrollment.EnrollmentChoice.AdmissionApplicationID == nil || *roadmapAfterEnrollment.EnrollmentChoice.AdmissionApplicationID != applicationID {
		t.Fatalf("roadmap must retain final enrollment choice: %v; choice=%#v", err, roadmapAfterEnrollment.EnrollmentChoice)
	}
	for index := 0; index < 5; index++ {
		current, err := roadmapService.GetRoadmap(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
		if err != nil || current.NextAction == nil {
			t.Fatalf("get pre-enrollment step %d: %v; roadmap=%#v", index, err, current)
		}
		if _, err := roadmapService.UpdateRoadmapStep(ctx, userID, dto.GetRoadmapStepRequest{RoadmapID: roadmap.ID, StepID: current.NextAction.ID}, dto.UpdateRoadmapStepRequest{Status: models.RoadmapStepStatusCompleted}); err != nil {
			t.Fatalf("complete pre-enrollment step %d: %v", index, err)
		}
	}
	confirmEnrollment, err := roadmapService.GetRoadmap(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
	if err != nil || confirmEnrollment.NextAction == nil || confirmEnrollment.NextAction.StepType != models.RoadmapStepTypeConfirmEnrollment {
		t.Fatalf("next step must be enrollment confirmation: %v; roadmap=%#v", err, confirmEnrollment)
	}
	if _, err := roadmapService.UpdateRoadmapStep(ctx, userID, dto.GetRoadmapStepRequest{RoadmapID: roadmap.ID, StepID: confirmEnrollment.NextAction.ID}, dto.UpdateRoadmapStepRequest{Status: models.RoadmapStepStatusCompleted}); err != nil {
		t.Fatalf("complete enrollment confirmation: %v", err)
	}
	learningRoadmap, err := roadmapService.GetRoadmap(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
	if err != nil || learningRoadmap.NextAction == nil || learningRoadmap.NextAction.StepType != models.RoadmapStepTypeLearnAtUniversity {
		t.Fatalf("next step must be learning: %v; roadmap=%#v", err, learningRoadmap)
	}
	opportunity, err := roadmapService.GetEmployerOpportunity(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
	if err != nil || !opportunity.IsAvailable || opportunity.Name == "" {
		t.Fatalf("get employer opportunity: %v; opportunity=%#v", err, opportunity)
	}
	for index := 0; index < 2; index++ {
		current, err := roadmapService.GetRoadmap(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
		if err != nil || current.NextAction == nil {
			t.Fatalf("get current roadmap step %d: %v; roadmap=%#v", index, err, current)
		}
		if _, err := roadmapService.UpdateRoadmapStep(ctx, userID, dto.GetRoadmapStepRequest{RoadmapID: roadmap.ID, StepID: current.NextAction.ID}, dto.UpdateRoadmapStepRequest{Status: models.RoadmapStepStatusCompleted}); err != nil {
			t.Fatalf("complete roadmap step %d: %v", index, err)
		}
	}
	application, err := roadmapService.SubmitEmployerApplication(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
	if err != nil || application.CompanyOpportunityID != opportunity.ID || application.Status != "submitted" {
		t.Fatalf("submit employer application: %v; application=%#v", err, application)
	}
	finalRoadmap, err := roadmapService.GetRoadmap(ctx, userID, dto.GetRoadmapRequest{RoadmapID: roadmap.ID})
	if err != nil || finalRoadmap.EmployerApplication == nil || finalRoadmap.EmployerApplication.Status != "submitted" {
		t.Fatalf("employer application must persist in roadmap: %v; roadmap=%#v", err, finalRoadmap)
	}
}

func assertEducationDiagnosis(t *testing.T, service services.AdmissionService, ctx context.Context, userID, roadmapID int64, want string) {
	t.Helper()
	diagnosis, err := service.DiagnoseEducationOptions(ctx, userID, dto.FindEducationOptionsRequest{RoadmapID: roadmapID})
	if err != nil {
		t.Fatalf("diagnose education options: %v", err)
	}
	if diagnosis.Reason != want {
		t.Fatalf("diagnosis reason = %q, want %q", diagnosis.Reason, want)
	}
}

func testSmallSurveyScenarioForNinthGrade(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileService, trajectoryService, roadmapService := scenarioServices(tx)
	ctx := context.Background()
	userID := scenarioUserID + 1

	company := companyByName(t, trajectoryService, ctx, "Т1")
	regionID := regionIDByName(t, db, "Пермский край")
	if _, err := profileService.SaveProfile(ctx, userID, dto.UpsertProfileRequest{Grade: 9, RegionID: regionID}); err != nil {
		t.Fatalf("save ninth-grade profile: %v", err)
	}
	if _, err := trajectoryService.SelectCompany(ctx, userID, dto.SelectCompanyRequest{CompanyID: company.ID}); err != nil {
		t.Fatalf("select company: %v", err)
	}
	directions, err := trajectoryService.GetCareerDirections(ctx, userID, dto.GetCareerDirectionsRequest{CompanyID: company.ID})
	if err != nil {
		t.Fatalf("get directions: %v", err)
	}
	direction := directionByName(t, directions, "Разработчик программного обеспечения")
	examSets, err := trajectoryService.GetRecommendedExamSets(ctx, dto.GetRecommendedExamSetsRequest{CareerDirectionID: direction.ID})
	if err != nil || len(examSets) == 0 {
		t.Fatalf("get EGE sets: %v; sets=%d", err, len(examSets))
	}
	inputs := make([]dto.UserSubjectInput, 0, len(examSets[0].ExamSubjectIDs))
	for _, id := range examSets[0].ExamSubjectIDs {
		inputs = append(inputs, dto.UserSubjectInput{ExamSubjectID: id, Status: models.SubjectStatusPlanned})
	}
	if _, err := profileService.SaveUserSubjects(ctx, userID, dto.SaveUserSubjectsRequest{Subjects: inputs}); err != nil {
		t.Fatalf("save planned EGE subjects: %v", err)
	}
	targetYear := expectedTargetAdmissionYear(9, time.Now())
	goal, err := trajectoryService.ConfirmGoal(ctx, userID, dto.ConfirmGoalRequest{CareerDirectionID: direction.ID, TargetAdmissionYear: targetYear})
	if err != nil {
		t.Fatalf("confirm goal: %v", err)
	}
	if goal.TargetAdmissionYear != targetYear {
		t.Fatalf("target admission year = %d, want %d", goal.TargetAdmissionYear, targetYear)
	}
	roadmap, err := roadmapService.CreateRoadmap(ctx, userID, dto.CreateRoadmapRequest{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("create roadmap: %v", err)
	}
	if len(roadmap.Steps) == 0 || roadmap.NextAction == nil {
		t.Fatal("ninth-grade roadmap must contain steps and next action")
	}
}

// TestSurveyServicesRejectInvalidTransitions фиксирует бизнес-ограничения
// до создания roadmap: нельзя использовать пустые или несуществующие идентификаторы,
// профиль вне аудитории 9–11 классов и цель без сохранённого профиля.
func TestSurveyServicesRejectInvalidTransitions(t *testing.T) {
	runScenario(t, "негативные переходы опроса", testSurveyServicesRejectInvalidTransitions)
}

func testSurveyServicesRejectInvalidTransitions(t *testing.T) {
	db := openScenarioDatabase(t)
	tx := beginScenarioTransaction(t, db)
	profileService, trajectoryService, roadmapService := scenarioServices(tx)
	ctx := context.Background()

	regionID := regionIDByName(t, db, "Пермский край")
	if _, err := profileService.SaveProfile(ctx, scenarioUserID+2, dto.UpsertProfileRequest{Grade: 8, RegionID: regionID}); err == nil {
		t.Fatal("profile for grade 8 must be rejected")
	}
	if _, err := trajectoryService.SelectCompany(ctx, scenarioUserID+2, dto.SelectCompanyRequest{}); err == nil {
		t.Fatal("empty company id must be rejected")
	}
	if _, err := trajectoryService.SelectCompany(ctx, scenarioUserID+2, dto.SelectCompanyRequest{CompanyID: 999999}); err == nil {
		t.Fatal("unknown company must be rejected")
	}
	if _, err := trajectoryService.GetCareerDirections(ctx, scenarioUserID+2, dto.GetCareerDirectionsRequest{}); err == nil {
		t.Fatal("empty company id for directions must be rejected")
	}
	if _, err := trajectoryService.GetRecommendedExamSets(ctx, dto.GetRecommendedExamSetsRequest{}); err == nil {
		t.Fatal("empty career direction id must be rejected")
	}
	if _, err := profileService.SaveUserSubjects(ctx, scenarioUserID+2, dto.SaveUserSubjectsRequest{Subjects: []dto.UserSubjectInput{{ExamSubjectID: 0, Status: models.SubjectStatusPlanned}}}); err == nil {
		t.Fatal("invalid EGE subject must be rejected")
	}

	var direction models.CareerDirection
	if err := tx.First(&direction).Error; err != nil {
		t.Fatalf("load demo direction: %v", err)
	}
	if _, err := trajectoryService.ConfirmGoal(ctx, scenarioUserID+2, dto.ConfirmGoalRequest{CareerDirectionID: direction.ID, TargetAdmissionYear: 2028}); err == nil {
		t.Fatal("goal without a saved profile must be rejected")
	}
	if _, err := roadmapService.CreateRoadmap(ctx, scenarioUserID+2, dto.CreateRoadmapRequest{}); err == nil {
		t.Fatal("roadmap without a goal must be rejected")
	}
}

func scenarioServices(tx *gorm.DB) (services.ProfileService, services.TrajectoryService, services.RoadmapService) {
	profileStore := repositories.NewGormProfileRepository(tx)
	careerStore := repositories.NewGormCareerRepository(tx)
	educationStore := repositories.NewGormEducationRepository(tx)
	roadmapStore := repositories.NewGormRoadmapRepository(tx)
	return services.NewProfileService(profileStore),
		services.NewTrajectoryService(careerStore, educationStore, profileStore, roadmapStore),
		services.NewRoadmapService(roadmapStore, careerStore)
}

func beginScenarioTransaction(t *testing.T, db *gorm.DB) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin scenario transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	return tx
}

func expectedTargetAdmissionYear(grade int16, now time.Time) int16 {
	year := now.Year()
	if now.Month() >= time.September {
		year++
	}
	return int16(year + int(11-grade))
}

func openScenarioDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1 and use the dedicated test database")
	}
	if os.Getenv("DB_NAME") != "employer_first_roadmap_test" {
		t.Fatalf("refusing to run scenario tests against DB_NAME=%q; use employer_first_roadmap_test", os.Getenv("DB_NAME"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := database.Open(ctx)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if err := database.SeedDemoData(db); err != nil {
		t.Fatalf("seed test database: %v", err)
	}
	return db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
}

func companyByName(t *testing.T, service services.TrajectoryService, ctx context.Context, name string) dto.CompanyCatalogItemResponse {
	t.Helper()
	companies, err := service.FindCompanies(ctx, dto.FindCompaniesRequest{Limit: 20})
	if err != nil {
		t.Fatalf("find companies: %v", err)
	}
	for _, company := range companies {
		if company.Name == name {
			return company
		}
	}
	t.Fatalf("company %q was not seeded", name)
	return dto.CompanyCatalogItemResponse{}
}

func mustDirections(t *testing.T, service services.TrajectoryService, ctx context.Context, userID, companyID int64) []dto.CareerDirectionResponse {
	t.Helper()
	directions, err := service.GetCareerDirections(ctx, userID, dto.GetCareerDirectionsRequest{CompanyID: companyID})
	if err != nil {
		t.Fatalf("get career directions: %v", err)
	}
	return directions
}

func regionIDByName(t *testing.T, db *gorm.DB, name string) int64 {
	t.Helper()
	var region models.Region
	if err := db.First(&region, "name = ?", name).Error; err != nil {
		t.Fatalf("find region %q: %v", name, err)
	}
	return region.ID
}

func directionByName(t *testing.T, directions []dto.CareerDirectionResponse, name string) dto.CareerDirectionResponse {
	t.Helper()
	for _, direction := range directions {
		if direction.Name == name {
			return direction
		}
	}
	t.Fatalf("direction %q was not returned", name)
	return dto.CareerDirectionResponse{}
}
