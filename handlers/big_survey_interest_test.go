package handlers

import (
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/utils"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestWelcomeMessageExplainsMainCommands(t *testing.T) {
	message := welcomeMessage()
	if !strings.HasPrefix(message, "Добро пожаловать!") || strings.Contains(message, "Employer First Roadmap") {
		t.Fatalf("welcome heading must be neutral: %q", message)
	}
	for _, command := range []string{"/start", "/roadmap", "/test_reminder", "/restart"} {
		if !strings.Contains(message, command) {
			t.Fatalf("welcome message must contain %s: %q", command, message)
		}
	}
}

func TestInterestOptionLabelsMatchTheMessageVariants(t *testing.T) {
	for index, profile := range surveyActivityProfiles {
		if got, want := numberedOptionLabel(index+1, profile.Label), fmt.Sprintf("%d. %s", index+1, profile.Label); got != want {
			t.Fatalf("activity button = %q, want %q", got, want)
		}
	}
	if got := numberedOptionLabel(6, "Пока не знаю"); got != "6. Пока не знаю" {
		t.Fatalf("unknown activity button = %q", got)
	}
}

func TestRecommendedCompanyLineDoesNotExposeRankingReason(t *testing.T) {
	line := recommendedCompanyLine(1, dto.RecommendedCompanyResponse{
		CompanyCatalogItemResponse: dto.CompanyCatalogItemResponse{
			Name:        "Тестовая компания",
			Description: "Короткое описание компании.",
		},
		Reasons: []string{"внутренняя причина"},
	})
	if strings.Contains(line, "Почему рекомендована") || strings.Contains(line, "внутренняя причина") {
		t.Fatalf("company line must not expose ranking reason: %q", line)
	}
}

func TestRoadmapExamAssessmentUsesExactEditedSet(t *testing.T) {
	s := &utils.SmallSurveyData{
		CareerDirectionID: 42,
		PlannedExamIDs:    []int64{1, 2, 3},
		SelectedExamIDs:   []int64{1, 4, 5},
	}
	request := roadmapExamSetAssessmentRequest(s)
	if request.CareerDirectionID != 42 || !slices.Equal(request.ExamSubjectIDs, []int64{1, 4, 5}) {
		t.Fatalf("assessment request = %#v, must use the exact edited set", request)
	}
}

func TestSurveyActivityProfilesUseBroadChoices(t *testing.T) {
	if len(surveyActivityProfiles) != 5 {
		t.Fatalf("activity profiles = %d, want 5", len(surveyActivityProfiles))
	}
	technology, ok := surveyActivityProfileByKey("technology")
	if !ok || technology.TagWeights["Программирование"] <= technology.TagWeights["Математика"] {
		t.Fatalf("technology profile must prioritize programming over supporting school tags: %#v", technology)
	}
	if _, ok := surveyActivityProfileByKey("unknown"); ok {
		t.Fatal("'Пока не знаю' must not add an interest profile")
	}
}

func TestSchoolSubjectsAreWeakIndependentSignals(t *testing.T) {
	math := schoolSubjectWeights("Математика")
	socialStudies := schoolSubjectWeights("Обществознание")
	russian := schoolSubjectWeights("Русский язык")

	if math["Математика"] >= 1 || socialStudies["Коммуникация"] >= 1 {
		t.Fatalf("school subject weights must stay weaker than primary activity weights: math=%#v social=%#v", math, socialStudies)
	}
	if math["Математика"] == 0 || socialStudies["Коммуникация"] == 0 {
		t.Fatalf("each selected subject must independently enrich the profile: math=%#v social=%#v", math, socialStudies)
	}
	if russian["Коммуникация"] == 0 || russian["Русский язык"] != 0 {
		t.Fatalf("Russian language must enrich humanities/communication, not represent an EGE constraint: %#v", russian)
	}
}

func TestExpectedScoreRangesUseUpperBounds(t *testing.T) {
	tests := map[string]int16{
		"80–100": 100,
		"60–79":  79,
		"0–59":   59,
	}
	for input, want := range tests {
		got, ok := expectedScoreRangeUpperBound(input)
		if !ok || got != want {
			t.Fatalf("range %q = (%d, %t), want (%d, true)", input, got, ok, want)
		}
	}
	if _, ok := expectedScoreRangeUpperBound("Пропустить"); ok {
		t.Fatal("skipping an expected EGE score must not be allowed")
	}
}

func TestCatalogOptionButtonKeepsShortTitles(t *testing.T) {
	if got, want := catalogOptionButton(6, "Госкорпорация «Росатом»"), "6. Госкорпорация «Росатом»"; got != want {
		t.Fatalf("short company button = %q, want %q", got, want)
	}
	if got, want := catalogOptionButton(1, "Системный аналитик"), "1. Системный аналитик"; got != want {
		t.Fatalf("short direction button = %q, want %q", got, want)
	}
	if got, want := catalogOptionButton(1, "Очень длинное название карьерного направления"), "1"; got != want {
		t.Fatalf("long catalog option button = %q, want %q", got, want)
	}
}

func TestTransitionFromSmallToBigSurveyClearsSelectedCompany(t *testing.T) {
	const userID int64 = 9_876_543
	s := utils.GetSmallSurvey(userID)
	s.CompanyID = 12
	s.CompanyName = "Компания из малого опроса"
	s.Grade = 10
	s.PlannedExamIDs = []int64{1, 2, 3}

	beginBigSurvey(userID)
	cleared := utils.GetSmallSurvey(userID)
	if cleared.CompanyID != 0 || cleared.CompanyName != "" || cleared.Grade != 0 || len(cleared.PlannedExamIDs) != 0 {
		t.Fatalf("small-survey data must be cleared before big survey: %#v", cleared)
	}
	if state := utils.GetUserState(userID); state != models.UserStateBigSurveyWaitingGrade {
		t.Fatalf("state = %q, want %q", state, models.UserStateBigSurveyWaitingGrade)
	}
}
