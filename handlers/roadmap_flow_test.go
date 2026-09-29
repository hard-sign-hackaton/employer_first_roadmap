package handlers

import (
	"errors"
	"strings"
	"testing"

	"efr_bot/dto"
	"efr_bot/models"
)

func TestEmployerOpportunityLinkPrefersDirectOpportunityURL(t *testing.T) {
	link := employerOpportunityLink(dto.CompanyOpportunityResponse{URL: "https://careers.example.test/intern", CompanyWebsiteURL: "https://example.test"})
	if !strings.Contains(link, "https://careers.example.test/intern") || strings.Contains(link, "https://example.test") {
		t.Fatalf("link = %q", link)
	}
}

func TestAdmissionPlanSaveErrorExplainsHowToCorrectChoice(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{errors.New("admission plan exceeds program limit for university 1"), "одном вузе"},
		{errors.New("admission plan exceeds university limit"), "слишком много вузов"},
		{errors.New("education program 1 is unavailable for current exam results"), "больше недоступен"},
		{errors.New("get latest admission campaign rule: record not found"), "Настройки приёмной кампании"},
	}
	for _, test := range tests {
		if message := admissionPlanSaveErrorMessage(test.err); !strings.Contains(message, test.want) {
			t.Fatalf("error %q: message %q must contain %q", test.err, message, test.want)
		}
	}
}

func TestRoadmapEmploymentActionIsShort(t *testing.T) {
	if action := roadmapPrimaryAction(models.RoadmapStepTypeApplyToEmployer); action != "Подать на работу" {
		t.Fatalf("employment action = %q", action)
	}
}

func TestEmployerOpportunityLinkFallsBackToCompanyWebsite(t *testing.T) {
	link := employerOpportunityLink(dto.CompanyOpportunityResponse{CompanyWebsiteURL: "https://example.test"})
	if !strings.Contains(link, "Подробнее о компании") || !strings.Contains(link, "https://example.test") {
		t.Fatalf("link = %q", link)
	}
}
