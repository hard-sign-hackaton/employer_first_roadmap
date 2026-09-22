package models

import (
	"time"

	"gorm.io/gorm"
)

type UserProfile struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement:false"`
	Grade     uint8     `json:"grade"`
	RegionID  uint      `json:"region_id"`
	CreatedAt time.Time `json:"created_at"`

	Region    Region         `json:"region"`
	Interests []UserInterest `json:"interests"`
	Subjects  []UserSubject  `json:"subjects"`
}

type Region struct {
	gorm.Model
	Name string `json:"name"`
}

type Interest struct {
	gorm.Model
	Name string `json:"name"`
}

type UserInterest struct {
	UserProfileID int64   `json:"user_id" gorm:"primaryKey"`
	InterestID    uint    `json:"interest_id" gorm:"primaryKey"`
	Weight        float64 `json:"weight"`

	Interest Interest `json:"interest"`
}

type Subject struct {
	gorm.Model
	Name string `json:"name"`
}

type UserSubject struct {
	UserProfileID int64  `json:"user_id" gorm:"primaryKey"`
	SubjectID     uint   `json:"subject_id" gorm:"primaryKey"`
	Status        string `json:"status"`
	Score         *uint  `json:"score"`

	Subject Subject `json:"subject"`
}
