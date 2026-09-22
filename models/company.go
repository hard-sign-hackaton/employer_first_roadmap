package models

import "gorm.io/gorm"

type Company struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`

	Vacancies []Vacancy `json:"vacancies"`
}
