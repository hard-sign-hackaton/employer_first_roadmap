package models

import "gorm.io/gorm"

type UserProfile struct {
	gorm.Model
	Grade  uint   `json:"grade"`
	Region string `json:"region"`

	Interests []Interest `json:"interests" gorm:"many2many:user_profile_interests"`
	Exams     []UserExam `json:"exams"`
}

type Interest struct {
	gorm.Model
	Name string `json:"name"`
}

type UserExam struct {
	gorm.Model
	UserProfileID uint  `json:"-"`
	ExamID        uint  `json:"exam_id"`
	Score         *uint `json:"score"`
	Planned       bool  `json:"planned"`

	Exam ExamType `json:"exam"`
}
