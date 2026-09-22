package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserProfile struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Grade     uint8     `json:"grade"`
	RegionID  uint      `json:"region_id"`
	CreatedAt time.Time `json:"created_at"`

	Region    Region         `json:"region"`
	Interests []UserInterest `json:"interests"`
	Subjects  []UserSubject  `json:"subjects"`
}

func (profile *UserProfile) BeforeCreate(_ *gorm.DB) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	return nil
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
	UserProfileID uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey"`
	InterestID    uint      `json:"interest_id" gorm:"primaryKey"`
	Weight        float64   `json:"weight"`

	Interest Interest `json:"interest"`
}

type Subject struct {
	gorm.Model
	Name string `json:"name"`
}

type UserSubject struct {
	UserProfileID uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey"`
	SubjectID     uint      `json:"subject_id" gorm:"primaryKey"`
	Status        string    `json:"status"`
	Score         *uint     `json:"score"`

	Subject Subject `json:"subject"`
}
