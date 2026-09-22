package models

import "gorm.io/gorm"

type EducationProgram struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`

	Requirements  []ExamRequirement
	Opportunities []Competence `gorm:"many2many:education_program_opportunities"`

	UniversityID uint
	University   University `gorm:"foreignKey:UniversityID"`
}

type ExamRequirement struct {
	gorm.Model

	MinScore       uint `json:"min_score"`
	MinRealScore   uint `json:"min_real_score"`
	MinBudgetScore uint `json:"min_budget_score"`

	Exam ExamType `gorm:"many2many:exam_requirement_types"`

	EducationProgramID uint `gorm:"foreignKey:EducationProgramID"`
}

type ExamType struct {
	gorm.Model

	Name string `json:"name"`
}
