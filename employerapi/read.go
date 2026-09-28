package employerapi

import (
	"context"
	"errors"
	"fmt"

	"efr_bot/models"
	"gorm.io/gorm"
)

func (s *Store) GetCatalog(ctx context.Context, companyID int64) (CatalogResponse, error) {
	var company models.Company
	if err := s.db.WithContext(ctx).First(&company, companyID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return CatalogResponse{}, fmt.Errorf("%w: company %d", ErrNotFound, companyID)
	} else if err != nil {
		return CatalogResponse{}, err
	}

	response := CatalogResponse{Company: CompanyResponse{ID: company.ID, Name: company.Name, Description: company.Description, WebsiteURL: company.WebsiteURL}}
	var directions []models.CareerDirection
	if err := s.db.WithContext(ctx).
		Preload("InterestTags.InterestTag").
		Preload("Opportunities.Region").
		Where("company_id = ?", companyID).
		Order("name ASC").Find(&directions).Error; err != nil {
		return CatalogResponse{}, err
	}

	response.Directions = make([]CareerDirectionResponse, 0, len(directions))
	for _, direction := range directions {
		directionResponse := CareerDirectionResponse{ID: direction.ID, Name: direction.Name, Description: direction.Description}
		directionResponse.InterestTags = make([]InterestTagResponse, 0, len(direction.InterestTags))
		for _, link := range direction.InterestTags {
			directionResponse.InterestTags = append(directionResponse.InterestTags, InterestTagResponse{ID: link.InterestTagID, Name: link.InterestTag.Name, Weight: link.Weight})
		}
		directionResponse.Opportunities = make([]OpportunityResponse, 0, len(direction.Opportunities))
		for _, opportunity := range direction.Opportunities {
			view := OpportunityResponse{ID: opportunity.ID, Type: opportunity.Type, Name: opportunity.Name, Description: opportunity.Description, URL: opportunity.URL, MinStudyYear: opportunity.MinStudyYear, RegionID: opportunity.RegionID, IsActive: opportunity.IsActive, UpdatedAt: opportunity.UpdatedAt}
			if opportunity.Region != nil {
				view.Region = opportunity.Region.Name
			}
			directionResponse.Opportunities = append(directionResponse.Opportunities, view)
		}

		programs, err := s.programsForDirection(ctx, direction.ID)
		if err != nil {
			return CatalogResponse{}, err
		}
		directionResponse.EducationPrograms = programs
		template, err := s.activeTemplate(ctx, direction.ID)
		if err != nil {
			return CatalogResponse{}, err
		}
		directionResponse.Roadmap = template
		response.Directions = append(response.Directions, directionResponse)
	}

	var rule models.AdmissionCampaignRule
	if err := s.db.WithContext(ctx).Order("admission_year DESC").First(&rule).Error; err == nil {
		response.AdmissionRule = &AdmissionRuleInput{AdmissionYear: rule.AdmissionYear, MaxUniversities: rule.MaxUniversities, MaxProgramsPerUniversity: rule.MaxProgramsPerUniversity}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return CatalogResponse{}, err
	}
	return response, nil
}

func (s *Store) programsForDirection(ctx context.Context, directionID int64) ([]EducationProgramResponse, error) {
	var programs []models.EducationProgram
	err := s.db.WithContext(ctx).
		Joins("JOIN career_direction_education_programs AS cdep ON cdep.education_program_id = education_programs.id").
		Where("cdep.career_direction_id = ?", directionID).
		Preload("University.Region").
		Preload("ExamCombinations", func(db *gorm.DB) *gorm.DB { return db.Order("admission_year DESC, id ASC") }).
		Preload("ExamCombinations.Items.ExamSubject").
		Preload("AdmissionScores", func(db *gorm.DB) *gorm.DB { return db.Order("admission_year DESC") }).
		Order("education_programs.name ASC").Find(&programs).Error
	if err != nil {
		return nil, err
	}
	result := make([]EducationProgramResponse, 0, len(programs))
	for _, program := range programs {
		view := EducationProgramResponse{
			ID: program.ID, Code: program.Code, Name: program.Name, Description: program.Description,
			University:       UniversityResponse{ID: program.University.ID, Name: program.University.Name, Description: program.University.Description, WebsiteURL: program.University.WebsiteURL, RegionID: program.University.RegionID, Region: program.University.Region.Name},
			ExamCombinations: make([]ExamCombinationView, 0, len(program.ExamCombinations)), AdmissionScores: make([]AdmissionScoreInput, 0, len(program.AdmissionScores)),
		}
		for _, combination := range program.ExamCombinations {
			combinationView := ExamCombinationView{ID: combination.ID, AdmissionYear: combination.AdmissionYear, Subjects: make([]ExamSubjectView, 0, len(combination.Items))}
			if combination.Name != nil {
				combinationView.Name = *combination.Name
			}
			for _, item := range combination.Items {
				combinationView.Subjects = append(combinationView.Subjects, ExamSubjectView{ID: item.ExamSubjectID, Name: item.ExamSubject.Name, MinScore: item.MinScore})
			}
			view.ExamCombinations = append(view.ExamCombinations, combinationView)
		}
		for _, score := range program.AdmissionScores {
			view.AdmissionScores = append(view.AdmissionScores, AdmissionScoreInput{AdmissionYear: score.AdmissionYear, BudgetPassingScore: score.BudgetPassingScore, PaidPassingScore: score.PaidPassingScore})
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Store) activeTemplate(ctx context.Context, directionID int64) (*RoadmapTemplateResponse, error) {
	var template models.RoadmapTemplate
	err := s.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("order_no ASC") }).
		Where("career_direction_id = ? AND is_active = ?", directionID, true).
		Order("version DESC").First(&template).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := &RoadmapTemplateResponse{ID: template.ID, Name: template.Name, Version: template.Version, IsActive: template.IsActive, Steps: make([]RoadmapStepInput, 0, len(template.Steps))}
	for _, step := range template.Steps {
		result.Steps = append(result.Steps, RoadmapStepInput{StepType: step.StepType, Title: step.Title, Description: step.Description})
	}
	return result, nil
}

func (s *Store) ReferenceData(ctx context.Context) (ReferenceDataResponse, error) {
	response := ReferenceDataResponse{OpportunityTypes: opportunityTypes, RoadmapStepTypes: make([]string, 0, len(canonicalRoadmapSteps))}
	for _, step := range canonicalRoadmapSteps {
		response.RoadmapStepTypes = append(response.RoadmapStepTypes, step.StepType)
	}
	var regions []models.Region
	var tags []models.InterestTag
	var subjects []models.ExamSubject
	if err := s.db.WithContext(ctx).Order("name ASC").Find(&regions).Error; err != nil {
		return response, err
	}
	if err := s.db.WithContext(ctx).Order("name ASC").Find(&tags).Error; err != nil {
		return response, err
	}
	if err := s.db.WithContext(ctx).Order("name ASC").Find(&subjects).Error; err != nil {
		return response, err
	}
	response.Regions = make([]NamedReference, 0, len(regions))
	response.InterestTags = make([]NamedReference, 0, len(tags))
	response.ExamSubjects = make([]NamedReference, 0, len(subjects))
	for _, region := range regions {
		response.Regions = append(response.Regions, NamedReference{ID: region.ID, Name: region.Name})
	}
	for _, tag := range tags {
		response.InterestTags = append(response.InterestTags, NamedReference{ID: tag.ID, Name: tag.Name})
	}
	for _, subject := range subjects {
		response.ExamSubjects = append(response.ExamSubjects, NamedReference{ID: subject.ID, Name: subject.Name})
	}
	return response, nil
}
