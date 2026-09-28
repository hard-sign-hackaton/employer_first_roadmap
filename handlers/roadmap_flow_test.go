package handlers

import (
	"strings"
	"testing"

	"efr_bot/dto"
)

func TestEmployerOpportunityLinkPrefersDirectOpportunityURL(t *testing.T) {
	link := employerOpportunityLink(dto.CompanyOpportunityResponse{URL: "https://careers.example.test/intern", CompanyWebsiteURL: "https://example.test"})
	if !strings.Contains(link, "https://careers.example.test/intern") || strings.Contains(link, "https://example.test") {
		t.Fatalf("link = %q", link)
	}
}

func TestEmployerOpportunityLinkFallsBackToCompanyWebsite(t *testing.T) {
	link := employerOpportunityLink(dto.CompanyOpportunityResponse{CompanyWebsiteURL: "https://example.test"})
	if !strings.Contains(link, "Подробнее о компании") || !strings.Contains(link, "https://example.test") {
		t.Fatalf("link = %q", link)
	}
}
