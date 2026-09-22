package models

import (
	"gorm.io/gorm"
)

type University struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`

	Programs []EducationProgram `json:"programs"`
}
