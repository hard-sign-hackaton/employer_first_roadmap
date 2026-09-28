package employerapi

import (
	"fmt"
	"net/url"
	"strings"

	"efr_bot/models"
)

var opportunityTypes = []string{
	models.OpportunityTypeInternship,
	models.OpportunityTypePractice,
	models.OpportunityTypeProject,
	models.OpportunityTypeHackathon,
	models.OpportunityTypeTargetedTraining,
}

var canonicalRoadmapSteps = []RoadmapStepInput{
	{StepType: models.RoadmapStepTypeChooseOrConfirmExams, Title: "Подтвердить набор ЕГЭ"},
	{StepType: models.RoadmapStepTypePrepareForExams, Title: "Подготовиться к ЕГЭ"},
	{StepType: models.RoadmapStepTypePassExams, Title: "Сдать ЕГЭ"},
	{StepType: models.RoadmapStepTypeChooseUniversity, Title: "Выбрать вузы и образовательные программы"},
	{StepType: models.RoadmapStepTypeSubmitAdmissionDocuments, Title: "Подать документы"},
	{StepType: models.RoadmapStepTypeConfirmEnrollment, Title: "Подтвердить зачисление"},
	{StepType: models.RoadmapStepTypeLearnAtUniversity, Title: "Учиться в выбранном вузе"},
	{StepType: models.RoadmapStepTypeEmployerExperience, Title: "Получить практический опыт у работодателя"},
	{StepType: models.RoadmapStepTypeApplyToEmployer, Title: "Подать заявку в компанию"},
}

func validateCatalog(request CatalogRequest, companyID int64) error {
	request.Company.Name = strings.TrimSpace(request.Company.Name)
	if request.Company.Name == "" {
		return fmt.Errorf("company.name is required")
	}
	if err := validateURL("company.website_url", request.Company.WebsiteURL); err != nil {
		return err
	}
	if len(request.Directions) == 0 {
		return fmt.Errorf("at least one direction is required")
	}

	directionNames := make(map[string]struct{}, len(request.Directions))
	for directionIndex, direction := range request.Directions {
		prefix := fmt.Sprintf("directions[%d]", directionIndex)
		nameKey := strings.ToLower(strings.TrimSpace(direction.Name))
		if nameKey == "" {
			return fmt.Errorf("%s.name is required", prefix)
		}
		if _, exists := directionNames[nameKey]; exists {
			return fmt.Errorf("%s.name is duplicated", prefix)
		}
		directionNames[nameKey] = struct{}{}
		if len(direction.InterestTags) == 0 {
			return fmt.Errorf("%s.interest_tags must not be empty", prefix)
		}
		if len(direction.EducationPrograms) == 0 {
			return fmt.Errorf("%s.education_programs must not be empty", prefix)
		}
		if len(direction.Opportunities) == 0 {
			return fmt.Errorf("%s.opportunities must not be empty", prefix)
		}
		for tagIndex, tag := range direction.InterestTags {
			if strings.TrimSpace(tag.Name) == "" || tag.Weight < 0 {
				return fmt.Errorf("%s.interest_tags[%d] must have a name and non-negative weight", prefix, tagIndex)
			}
		}
		for programIndex, program := range direction.EducationPrograms {
			programPrefix := fmt.Sprintf("%s.education_programs[%d]", prefix, programIndex)
			if strings.TrimSpace(program.Name) == "" || strings.TrimSpace(program.University.Name) == "" || strings.TrimSpace(program.University.Region) == "" {
				return fmt.Errorf("%s name, university.name and university.region are required", programPrefix)
			}
			if err := validateURL(programPrefix+".university.website_url", program.University.WebsiteURL); err != nil {
				return err
			}
			if len(program.ExamCombinations) == 0 {
				return fmt.Errorf("%s.exam_combinations must not be empty", programPrefix)
			}
			for combinationIndex, combination := range program.ExamCombinations {
				if combination.AdmissionYear < 2020 || combination.AdmissionYear > 2100 || len(combination.Subjects) == 0 {
					return fmt.Errorf("%s.exam_combinations[%d] needs a valid admission_year and subjects", programPrefix, combinationIndex)
				}
				for subjectIndex, subject := range combination.Subjects {
					if strings.TrimSpace(subject.Name) == "" || (subject.MinScore != nil && (*subject.MinScore < 0 || *subject.MinScore > 100)) {
						return fmt.Errorf("%s.exam_combinations[%d].subjects[%d] is invalid", programPrefix, combinationIndex, subjectIndex)
					}
				}
			}
		}
		for opportunityIndex, opportunity := range direction.Opportunities {
			opportunityPrefix := fmt.Sprintf("%s.opportunities[%d]", prefix, opportunityIndex)
			if strings.TrimSpace(opportunity.Name) == "" || !contains(opportunityTypes, opportunity.Type) {
				return fmt.Errorf("%s needs a name and supported type", opportunityPrefix)
			}
			if opportunity.MinStudyYear < 1 || opportunity.MinStudyYear > 6 {
				return fmt.Errorf("%s.min_study_year must be between 1 and 6", opportunityPrefix)
			}
			if err := validateURL(opportunityPrefix+".url", opportunity.URL); err != nil {
				return err
			}
		}
		if direction.Roadmap != nil && len(direction.Roadmap.Steps) > 0 {
			if err := validateRoadmapSteps(prefix+".roadmap.steps", direction.Roadmap.Steps); err != nil {
				return err
			}
		}
	}
	if companyID == 0 && request.AdmissionRule == nil {
		return fmt.Errorf("admission_rule is required when creating a catalog")
	}
	if request.AdmissionRule != nil {
		rule := request.AdmissionRule
		if rule.AdmissionYear < 2020 || rule.AdmissionYear > 2100 || rule.MaxUniversities <= 0 || rule.MaxProgramsPerUniversity <= 0 {
			return fmt.Errorf("admission_rule is invalid")
		}
	}
	return nil
}

func validateRoadmapSteps(path string, steps []RoadmapStepInput) error {
	if len(steps) != len(canonicalRoadmapSteps) {
		return fmt.Errorf("%s must contain all %d bot steps", path, len(canonicalRoadmapSteps))
	}
	for index, expected := range canonicalRoadmapSteps {
		if steps[index].StepType != expected.StepType || strings.TrimSpace(steps[index].Title) == "" {
			return fmt.Errorf("%s[%d] must be %q and have a title", path, index, expected.StepType)
		}
	}
	return nil
}

func validateURL(path, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute http(s) URL", path)
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
