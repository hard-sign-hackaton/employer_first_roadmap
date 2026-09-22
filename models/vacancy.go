package models

import "gorm.io/gorm"

type Vacancy struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`

	Requirements []Competence `json:"requirements"`
	Company      Company      `json:"company"`
}

type Competence struct {
	gorm.Model
	Name string `json:"name"`
}
