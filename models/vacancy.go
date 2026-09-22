package models

import "gorm.io/gorm"

type Vacancy struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`

	Requirements []Competence `gorm:"many2many:vacancy_requirements"`

	Company   Company `gorm:"foreignKey:CompanyID"`
	CompanyID uint
}

type Competence struct {
	gorm.Model
	Name string `json:"name"`
}
