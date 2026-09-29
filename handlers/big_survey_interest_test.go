package handlers

import (
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
