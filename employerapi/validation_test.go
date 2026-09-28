package employerapi

import (
	"strings"
	"testing"
)

func TestValidateCatalogAcceptsCompleteEmployerData(t *testing.T) {
	request := validCatalogRequest()
	if err := validateCatalog(request, 0); err != nil {
		t.Fatalf("validate complete catalog: %v", err)
	}
}

func TestValidateCatalogRequiresDataUsedByBot(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*CatalogRequest)
		message string
	}{
		{"company", func(request *CatalogRequest) { request.Company.Name = "" }, "company.name"},
		{"admission rule", func(request *CatalogRequest) { request.AdmissionRule = nil }, "admission_rule"},
		{"programs", func(request *CatalogRequest) { request.Directions[0].EducationPrograms = nil }, "education_programs"},
		{"opportunities", func(request *CatalogRequest) { request.Directions[0].Opportunities = nil }, "opportunities"},
		{"exam subjects", func(request *CatalogRequest) {
			request.Directions[0].EducationPrograms[0].ExamCombinations[0].Subjects = nil
		}, "subjects"},
		{"roadmap order", func(request *CatalogRequest) {
			request.Directions[0].Roadmap.Steps[0], request.Directions[0].Roadmap.Steps[1] = request.Directions[0].Roadmap.Steps[1], request.Directions[0].Roadmap.Steps[0]
		}, "must be"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validCatalogRequest()
			test.change(&request)
			err := validateCatalog(request, 0)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want message containing %q", err, test.message)
			}
		})
	}
}

func validCatalogRequest() CatalogRequest {
	active := true
	minimum := int16(40)
	steps := append([]RoadmapStepInput(nil), canonicalRoadmapSteps...)
	return CatalogRequest{
		Company:       CompanyInput{Name: "Работодатель", WebsiteURL: "https://example.com"},
		AdmissionRule: &AdmissionRuleInput{AdmissionYear: 2027, MaxUniversities: 5, MaxProgramsPerUniversity: 5},
		Directions: []CareerDirectionInput{{
			Name:         "Разработчик",
			InterestTags: []InterestTagInput{{Name: "Программирование", Weight: 1}},
			EducationPrograms: []EducationProgramInput{{
				Name:             "Программная инженерия",
				University:       UniversityInput{Name: "Университет", Region: "Пермский край", WebsiteURL: "https://university.example.com"},
				ExamCombinations: []ExamCombinationInput{{AdmissionYear: 2027, Subjects: []ExamSubjectInput{{Name: "Русский язык", MinScore: &minimum}}}},
			}},
			Opportunities: []OpportunityInput{{Type: "internship", Name: "Стажировка", URL: "https://example.com/internship", MinStudyYear: 2, IsActive: &active}},
			Roadmap:       &RoadmapTemplateInput{Name: "Путь разработчика", Steps: steps},
		}},
	}
}
