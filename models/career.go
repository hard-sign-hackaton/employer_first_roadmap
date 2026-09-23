package models

import "time"

// Company — работодатель, выбранный напрямую или рекомендованный после полного опроса.
type Company struct {
	ID          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"size:200;not null;unique"`
	Description string `json:"description"`

	CareerDirections []CareerDirection    `json:"career_directions" gorm:"foreignKey:CompanyID"`
	Opportunities    []CompanyOpportunity `json:"opportunities" gorm:"foreignKey:CompanyID"`
}

// CareerDirection — конкретное семейство ролей в компании и цель пользователя.
type CareerDirection struct {
	ID          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID   int64  `json:"company_id" gorm:"not null;uniqueIndex:ux_career_direction_per_company"`
	Name        string `json:"name" gorm:"size:200;not null;uniqueIndex:ux_career_direction_per_company"`
	Description string `json:"description"`

	Company       Company                      `json:"company" gorm:"foreignKey:CompanyID"`
	InterestTags  []CareerDirectionInterestTag `json:"interest_tags" gorm:"foreignKey:CareerDirectionID"`
	Opportunities []CompanyOpportunity         `json:"opportunities" gorm:"foreignKey:CareerDirectionID"`
}

// CareerDirectionInterestTag делает возможным детерминированное ранжирование по интересам.
type CareerDirectionInterestTag struct {
	CareerDirectionID int64   `json:"career_direction_id" gorm:"primaryKey"`
	InterestTagID     int64   `json:"interest_tag_id" gorm:"primaryKey"`
	Weight            float64 `json:"weight" gorm:"not null;check:weight >= 0"`

	InterestTag InterestTag `json:"interest_tag" gorm:"foreignKey:InterestTagID"`
}

// CompanyOpportunity — вручную настроенный переход от обучения в вузе к работодателю.
type CompanyOpportunity struct {
	ID                int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID         int64     `json:"company_id" gorm:"not null;index"`
	CareerDirectionID int64     `json:"career_direction_id" gorm:"not null;index"`
	Type              string    `json:"type" gorm:"size:24;not null;check:type IN ('internship','practice','project','hackathon','targeted_training')"`
	Name              string    `json:"name" gorm:"size:255;not null"`
	Description       string    `json:"description"`
	URL               string    `json:"url"`
	MinStudyYear      int16     `json:"min_study_year" gorm:"not null;check:min_study_year BETWEEN 1 AND 6"`
	RegionID          *int64    `json:"region_id"`
	IsActive          bool      `json:"is_active" gorm:"not null;default:true"`
	UpdatedAt         time.Time `json:"updated_at"`

	Company         Company         `json:"company" gorm:"foreignKey:CompanyID"`
	CareerDirection CareerDirection `json:"career_direction" gorm:"foreignKey:CareerDirectionID"`
	Region          *Region         `json:"region,omitempty" gorm:"foreignKey:RegionID"`
}
