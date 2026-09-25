package repositories

import (
	"context"

	"efr_bot/models"
	"efr_bot/services/ports"
	"gorm.io/gorm"
)

// GormEducationRepository реализует EducationRepository через GORM.
type GormEducationRepository struct {
	db *gorm.DB
}

var _ ports.EducationStore = (*GormEducationRepository)(nil)

// NewGormEducationRepository создаёт репозиторий образования на переданном подключении GORM.
func NewGormEducationRepository(db *gorm.DB) *GormEducationRepository {
	return &GormEducationRepository{db: db}
}

func (r *GormEducationRepository) ListLatestExamCombinationsForDirection(ctx context.Context, careerDirectionID int64) ([]models.ExamCombination, error) {
	var combinations []models.ExamCombination
	latestYear := r.db.Model(&models.ExamCombination{}).
		Select("MAX(exam_combinations.admission_year)").
		Joins("JOIN career_direction_education_programs AS cdep ON cdep.education_program_id = exam_combinations.education_program_id").
		Where("cdep.career_direction_id = ?", careerDirectionID)
	err := r.db.WithContext(ctx).
		Joins("JOIN career_direction_education_programs AS cdep ON cdep.education_program_id = exam_combinations.education_program_id").
		Where("cdep.career_direction_id = ? AND exam_combinations.admission_year = (?)", careerDirectionID, latestYear).
		Preload("EducationProgram.University.Region").
		Preload("Items.ExamSubject").
		Order("exam_combinations.id ASC").
		Find(&combinations).Error
	return combinations, err
}

func (r *GormEducationRepository) ListEducationOptions(ctx context.Context, filter ports.EducationOptionsFilter) ([]models.EducationProgram, error) {
	optionsQuery := r.db.WithContext(ctx).
		Model(&models.EducationProgram{}).
		Joins("JOIN career_direction_education_programs AS cdep ON cdep.education_program_id = education_programs.id").
		Joins("JOIN exam_combinations AS ec ON ec.education_program_id = education_programs.id").
		Joins("JOIN universities ON universities.id = education_programs.university_id").
		Where("cdep.career_direction_id = ?", filter.CareerDirectionID).
		Where(`ec.admission_year = (
			SELECT MAX(ec_latest.admission_year)
			FROM exam_combinations AS ec_latest
			WHERE ec_latest.education_program_id = education_programs.id
		)`)

	if !filter.ExpandGeography {
		optionsQuery = optionsQuery.Where("universities.region_id = ?", filter.RegionID)
	}
	if len(filter.ExamSubjectIDs) > 0 {
		optionsQuery = optionsQuery.Where(
			"NOT EXISTS (SELECT 1 FROM exam_combination_items AS eci WHERE eci.exam_combination_id = ec.id AND eci.exam_subject_id NOT IN ?)",
			filter.ExamSubjectIDs,
		)
	}

	var programs []models.EducationProgram
	err := optionsQuery.
		Distinct("education_programs.*").
		Preload("University.Region").
		Preload("ExamCombinations").
		Preload("ExamCombinations.Items.ExamSubject").
		Preload("AdmissionScores").
		Order("education_programs.name ASC").
		Find(&programs).Error
	return programs, err
}

func (r *GormEducationRepository) ListDirectionIDsAvailableForSubjects(ctx context.Context, companyID int64, examSubjectIDs []int64) ([]int64, error) {
	if len(examSubjectIDs) == 0 {
		return []int64{}, nil
	}

	var directionIDs []int64
	err := r.db.WithContext(ctx).
		Table("career_directions AS cd").
		Select("DISTINCT cd.id").
		Joins("JOIN career_direction_education_programs AS cdep ON cdep.career_direction_id = cd.id").
		Joins("JOIN exam_combinations AS ec ON ec.education_program_id = cdep.education_program_id").
		Where("cd.company_id = ? AND ec.admission_year = (SELECT MAX(ec_latest.admission_year) FROM exam_combinations AS ec_latest WHERE ec_latest.education_program_id = ec.education_program_id)", companyID).
		Where(
			"NOT EXISTS (SELECT 1 FROM exam_combination_items AS eci WHERE eci.exam_combination_id = ec.id AND eci.exam_subject_id NOT IN ?)",
			examSubjectIDs,
		).
		Order("cd.id ASC").
		Scan(&directionIDs).Error
	return directionIDs, err
}

func (r *GormEducationRepository) GetLatestAdmissionCampaignRule(ctx context.Context) (models.AdmissionCampaignRule, error) {
	var rule models.AdmissionCampaignRule
	err := r.db.WithContext(ctx).
		Order("admission_year DESC").
		First(&rule).Error
	return rule, err
}
