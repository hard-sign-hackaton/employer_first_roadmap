package models

import "gorm.io/gorm"

type EducationProgram struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`

	Requirements  []ExamRequirement `json:"requirements"`
	Opportunities []Competence      `json:"opportunities"`
	University    University        `json:"university"`
}

type ExamRequirement struct {
	gorm.Model

	MinScore       uint `json:"min_score"`
	MinRealScore   uint `json:"min_real_score"`
	MinBudgetScore uint `json:"min_budget_score"`

	Exam ExamType `json:"exam"`
}

type ExamType struct {
	gorm.Model

	Name string `json:"name"`
}
