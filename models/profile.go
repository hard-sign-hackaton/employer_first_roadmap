package models

import "time"

// Region — субъект РФ, используемый для места проживания школьника и расположения вуза.
type Region struct {
	ID   int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"size:120;not null;unique"`
}

// UserProfile хранит минимальный профиль, собираемый в обоих входных опросах.
type UserProfile struct {
	ID                int64     `json:"id" gorm:"primaryKey;autoIncrement:false"`
	Grade             int16     `json:"grade" gorm:"not null;check:grade BETWEEN 9 AND 11"`
	RegionID          int64     `json:"region_id" gorm:"not null;index"`
	WillingToRelocate bool      `json:"willing_to_relocate" gorm:"not null;default:false"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Region    Region         `json:"region" gorm:"foreignKey:RegionID"`
	Interests []UserInterest `json:"interests" gorm:"foreignKey:UserProfileID"`
	Subjects  []UserSubject  `json:"subjects" gorm:"foreignKey:UserProfileID"`
	Goals     []UserGoal     `json:"goals" gorm:"foreignKey:UserProfileID"`
}

// InterestTag используется и для ответов полного опроса, и для меток карьерных направлений.
type InterestTag struct {
	ID   int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"size:120;not null;unique"`
}

// UserInterest — нормализованный результат полного опроса.
type UserInterest struct {
	UserProfileID int64   `json:"user_profile_id" gorm:"primaryKey"`
	InterestTagID int64   `json:"interest_tag_id" gorm:"primaryKey"`
	Weight        float64 `json:"weight" gorm:"not null;check:weight >= 0"`

	InterestTag InterestTag `json:"interest_tag" gorm:"foreignKey:InterestTagID"`
}

// ExamSubject — справочник предметов ЕГЭ. Количество выбранных предметов намеренно не ограничено.
type ExamSubject struct {
	ID   int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"size:120;not null;unique"`
}

// UserSubject отражает путь от планируемого ЕГЭ к выбранному и сданному.
type UserSubject struct {
	UserProfileID int64  `json:"user_profile_id" gorm:"primaryKey"`
	ExamSubjectID int64  `json:"exam_subject_id" gorm:"primaryKey"`
	Status        string `json:"status" gorm:"size:16;not null;check:status IN ('planned','selected','passed')"`
	ExpectedScore *int16 `json:"expected_score" gorm:"check:expected_score BETWEEN 0 AND 100"`
	ActualScore   *int16 `json:"actual_score" gorm:"check:actual_score BETWEEN 0 AND 100"`

	ExamSubject ExamSubject `json:"exam_subject" gorm:"foreignKey:ExamSubjectID"`
}
