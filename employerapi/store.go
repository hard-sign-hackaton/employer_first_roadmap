package employerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"efr_bot/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SaveCatalog(ctx context.Context, companyID int64, request CatalogRequest) (CatalogResponse, error) {
	var savedCompanyID int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		company, err := saveCompany(tx, companyID, request.Company)
		if err != nil {
			return err
		}
		savedCompanyID = company.ID
		// A submitted catalog is the source of truth for visible opportunities.
		// Entries omitted from an update remain for historical roadmaps, but are
		// no longer offered to new users.
		if err := tx.Model(&models.CompanyOpportunity{}).Where("company_id = ?", company.ID).Update("is_active", false).Error; err != nil {
			return fmt.Errorf("deactivate previous opportunities: %w", err)
		}

		for _, input := range request.Directions {
			direction, err := saveDirection(tx, company.ID, input)
			if err != nil {
				return err
			}
			if err := replaceInterestTags(tx, direction.ID, input.InterestTags); err != nil {
				return err
			}
			if err := tx.Where("career_direction_id = ?", direction.ID).Delete(&models.CareerDirectionEducationProgram{}).Error; err != nil {
				return fmt.Errorf("replace direction education programs: %w", err)
			}
			for _, programInput := range input.EducationPrograms {
				program, err := saveEducationProgram(tx, programInput)
				if err != nil {
					return err
				}
				link := models.CareerDirectionEducationProgram{CareerDirectionID: direction.ID, EducationProgramID: program.ID}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
					return fmt.Errorf("link education program: %w", err)
				}
			}
			for _, opportunityInput := range input.Opportunities {
				if _, err := saveOpportunity(tx, company.ID, direction.ID, opportunityInput); err != nil {
					return err
				}
			}

			roadmapInput := input.Roadmap
			if companyID == 0 && roadmapInput == nil {
				roadmapInput = &RoadmapTemplateInput{Name: "Roadmap: " + direction.Name, Steps: canonicalRoadmapSteps}
			}
			if roadmapInput != nil {
				if len(roadmapInput.Steps) == 0 {
					roadmapInput.Steps = canonicalRoadmapSteps
				}
				if err := saveRoadmapTemplate(tx, direction.ID, *roadmapInput); err != nil {
					return err
				}
			}
		}

		if request.AdmissionRule != nil {
			rule := models.AdmissionCampaignRule{
				AdmissionYear: request.AdmissionRule.AdmissionYear, MaxUniversities: request.AdmissionRule.MaxUniversities,
				MaxProgramsPerUniversity: request.AdmissionRule.MaxProgramsPerUniversity,
			}
			if err := tx.Save(&rule).Error; err != nil {
				return fmt.Errorf("save admission rule: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return CatalogResponse{}, err
	}
	return s.GetCatalog(ctx, savedCompanyID)
}

func saveCompany(tx *gorm.DB, companyID int64, input CompanyInput) (models.Company, error) {
	if companyID == 0 {
		var count int64
		if err := tx.Model(&models.Company{}).Where("LOWER(name) = LOWER(?)", strings.TrimSpace(input.Name)).Count(&count).Error; err != nil {
			return models.Company{}, err
		}
		if count > 0 {
			return models.Company{}, fmt.Errorf("%w: company %q already exists", ErrConflict, input.Name)
		}
		company := models.Company{Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description), WebsiteURL: strings.TrimSpace(input.WebsiteURL)}
		if err := tx.Create(&company).Error; err != nil {
			return models.Company{}, fmt.Errorf("create company: %w", err)
		}
		return company, nil
	}

	var company models.Company
	if err := tx.First(&company, companyID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Company{}, fmt.Errorf("%w: company %d", ErrNotFound, companyID)
	} else if err != nil {
		return models.Company{}, err
	}
	company.Name = strings.TrimSpace(input.Name)
	company.Description = strings.TrimSpace(input.Description)
	company.WebsiteURL = strings.TrimSpace(input.WebsiteURL)
	if err := tx.Save(&company).Error; err != nil {
		return models.Company{}, fmt.Errorf("update company: %w", err)
	}
	return company, nil
}

func saveDirection(tx *gorm.DB, companyID int64, input CareerDirectionInput) (models.CareerDirection, error) {
	var direction models.CareerDirection
	err := tx.Where("company_id = ? AND LOWER(name) = LOWER(?)", companyID, strings.TrimSpace(input.Name)).First(&direction).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		direction = models.CareerDirection{CompanyID: companyID, Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description)}
		err = tx.Create(&direction).Error
	} else if err == nil {
		err = tx.Model(&direction).Updates(map[string]any{"name": strings.TrimSpace(input.Name), "description": strings.TrimSpace(input.Description)}).Error
	}
	if err != nil {
		return models.CareerDirection{}, fmt.Errorf("save direction %q: %w", input.Name, err)
	}
	return direction, nil
}

func replaceInterestTags(tx *gorm.DB, directionID int64, inputs []InterestTagInput) error {
	if err := tx.Where("career_direction_id = ?", directionID).Delete(&models.CareerDirectionInterestTag{}).Error; err != nil {
		return fmt.Errorf("replace direction tags: %w", err)
	}
	for _, input := range inputs {
		tag := models.InterestTag{Name: strings.TrimSpace(input.Name)}
		if err := tx.Where("LOWER(name) = LOWER(?)", tag.Name).FirstOrCreate(&tag).Error; err != nil {
			return fmt.Errorf("save interest tag %q: %w", input.Name, err)
		}
		link := models.CareerDirectionInterestTag{CareerDirectionID: directionID, InterestTagID: tag.ID, Weight: input.Weight}
		if err := tx.Create(&link).Error; err != nil {
			return fmt.Errorf("link interest tag %q: %w", input.Name, err)
		}
	}
	return nil
}

func saveEducationProgram(tx *gorm.DB, input EducationProgramInput) (models.EducationProgram, error) {
	region := models.Region{Name: strings.TrimSpace(input.University.Region)}
	if err := tx.Where("LOWER(name) = LOWER(?)", region.Name).FirstOrCreate(&region).Error; err != nil {
		return models.EducationProgram{}, fmt.Errorf("save region %q: %w", region.Name, err)
	}

	var university models.University
	err := tx.Where("region_id = ? AND LOWER(name) = LOWER(?)", region.ID, strings.TrimSpace(input.University.Name)).First(&university).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		university = models.University{RegionID: region.ID, Name: strings.TrimSpace(input.University.Name)}
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.EducationProgram{}, fmt.Errorf("find university: %w", err)
	}
	university.Description = strings.TrimSpace(input.University.Description)
	university.WebsiteURL = strings.TrimSpace(input.University.WebsiteURL)
	if university.ID == 0 {
		err = tx.Create(&university).Error
	} else {
		err = tx.Save(&university).Error
	}
	if err != nil {
		return models.EducationProgram{}, fmt.Errorf("save university %q: %w", university.Name, err)
	}

	var program models.EducationProgram
	programQuery := tx.Where("university_id = ?", university.ID)
	if strings.TrimSpace(input.Code) != "" {
		programQuery = programQuery.Where("code = ?", strings.TrimSpace(input.Code))
	} else {
		programQuery = programQuery.Where("LOWER(name) = LOWER(?)", strings.TrimSpace(input.Name))
	}
	err = programQuery.First(&program).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		program = models.EducationProgram{UniversityID: university.ID}
	} else if err != nil {
		return models.EducationProgram{}, fmt.Errorf("find education program: %w", err)
	}
	program.Code = strings.TrimSpace(input.Code)
	program.Name = strings.TrimSpace(input.Name)
	program.Description = strings.TrimSpace(input.Description)
	if program.ID == 0 {
		err = tx.Create(&program).Error
	} else {
		err = tx.Save(&program).Error
	}
	if err != nil {
		return models.EducationProgram{}, fmt.Errorf("save education program %q: %w", program.Name, err)
	}

	for _, combinationInput := range input.ExamCombinations {
		if err := saveExamCombination(tx, program.ID, combinationInput); err != nil {
			return models.EducationProgram{}, err
		}
	}
	for _, scoreInput := range input.AdmissionScores {
		score := models.AdmissionScoreHistory{EducationProgramID: program.ID, AdmissionYear: scoreInput.AdmissionYear, BudgetPassingScore: scoreInput.BudgetPassingScore, PaidPassingScore: scoreInput.PaidPassingScore}
		if err := tx.Save(&score).Error; err != nil {
			return models.EducationProgram{}, fmt.Errorf("save admission score: %w", err)
		}
	}
	return program, nil
}

func saveExamCombination(tx *gorm.DB, programID int64, input ExamCombinationInput) error {
	name := strings.TrimSpace(input.Name)
	var combination models.ExamCombination
	query := tx.Where("education_program_id = ? AND admission_year = ?", programID, input.AdmissionYear)
	if name == "" {
		query = query.Where("name IS NULL")
	} else {
		query = query.Where("name = ?", name)
	}
	err := query.First(&combination).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		combination = models.ExamCombination{EducationProgramID: programID, AdmissionYear: input.AdmissionYear}
		if name != "" {
			combination.Name = &name
		}
		if err := tx.Create(&combination).Error; err != nil {
			return fmt.Errorf("create exam combination: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find exam combination: %w", err)
	}
	if err := tx.Where("exam_combination_id = ?", combination.ID).Delete(&models.ExamCombinationItem{}).Error; err != nil {
		return fmt.Errorf("replace exam combination subjects: %w", err)
	}
	for _, subjectInput := range input.Subjects {
		subject := models.ExamSubject{Name: strings.TrimSpace(subjectInput.Name)}
		if err := tx.Where("LOWER(name) = LOWER(?)", subject.Name).FirstOrCreate(&subject).Error; err != nil {
			return fmt.Errorf("save exam subject %q: %w", subject.Name, err)
		}
		item := models.ExamCombinationItem{ExamCombinationID: combination.ID, ExamSubjectID: subject.ID, MinScore: subjectInput.MinScore}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("save exam subject %q: %w", subject.Name, err)
		}
	}
	return nil
}

func saveOpportunity(tx *gorm.DB, companyID, directionID int64, input OpportunityInput) (models.CompanyOpportunity, error) {
	var opportunity models.CompanyOpportunity
	var err error
	if input.ID > 0 {
		err = tx.Where("id = ? AND company_id = ? AND career_direction_id = ?", input.ID, companyID, directionID).First(&opportunity).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return opportunity, fmt.Errorf("%w: opportunity %d", ErrNotFound, input.ID)
		}
	} else {
		err = tx.Where("company_id = ? AND career_direction_id = ? AND LOWER(name) = LOWER(?)", companyID, directionID, strings.TrimSpace(input.Name)).First(&opportunity).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			opportunity = models.CompanyOpportunity{CompanyID: companyID, CareerDirectionID: directionID}
			err = nil
		}
	}
	if err != nil {
		return opportunity, fmt.Errorf("find opportunity: %w", err)
	}
	var regionID *int64
	if strings.TrimSpace(input.Region) != "" {
		region := models.Region{Name: strings.TrimSpace(input.Region)}
		if err := tx.Where("LOWER(name) = LOWER(?)", region.Name).FirstOrCreate(&region).Error; err != nil {
			return opportunity, fmt.Errorf("save opportunity region: %w", err)
		}
		regionID = &region.ID
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	opportunity.Type = input.Type
	opportunity.Name = strings.TrimSpace(input.Name)
	opportunity.Description = strings.TrimSpace(input.Description)
	opportunity.URL = strings.TrimSpace(input.URL)
	opportunity.MinStudyYear = input.MinStudyYear
	opportunity.RegionID = regionID
	opportunity.IsActive = isActive
	if opportunity.ID == 0 {
		err = tx.Create(&opportunity).Error
	} else {
		err = tx.Save(&opportunity).Error
	}
	if err != nil {
		return opportunity, fmt.Errorf("save opportunity %q: %w", opportunity.Name, err)
	}
	return opportunity, nil
}

func saveRoadmapTemplate(tx *gorm.DB, directionID int64, input RoadmapTemplateInput) error {
	var version int
	if err := tx.Model(&models.RoadmapTemplate{}).Where("career_direction_id = ?", directionID).Select("COALESCE(MAX(version), 0)").Scan(&version).Error; err != nil {
		return fmt.Errorf("read roadmap version: %w", err)
	}
	if err := tx.Model(&models.RoadmapTemplate{}).Where("career_direction_id = ? AND is_active = ?", directionID, true).Update("is_active", false).Error; err != nil {
		return fmt.Errorf("deactivate previous roadmap template: %w", err)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "Roadmap работодателя"
	}
	template := models.RoadmapTemplate{CareerDirectionID: directionID, Name: name, Version: version + 1, IsActive: true}
	if err := tx.Create(&template).Error; err != nil {
		return fmt.Errorf("create roadmap template: %w", err)
	}
	for index, stepInput := range input.Steps {
		step := models.RoadmapTemplateStep{RoadmapTemplateID: template.ID, OrderNo: int16(index + 1), StepType: stepInput.StepType, Title: strings.TrimSpace(stepInput.Title), Description: strings.TrimSpace(stepInput.Description)}
		if err := tx.Create(&step).Error; err != nil {
			return fmt.Errorf("create roadmap step %d: %w", index+1, err)
		}
	}
	return nil
}

func (s *Store) SetOpportunityStatus(ctx context.Context, opportunityID int64, active bool) error {
	result := s.db.WithContext(ctx).Model(&models.CompanyOpportunity{}).Where("id = ?", opportunityID).Update("is_active", active)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: opportunity %d", ErrNotFound, opportunityID)
	}
	return nil
}
