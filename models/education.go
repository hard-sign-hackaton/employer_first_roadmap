package models

// University находится в одном регионе и содержит образовательные программы.
type University struct {
	ID          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	RegionID    int64  `json:"region_id" gorm:"not null;uniqueIndex:ux_university_per_region"`
	Name        string `json:"name" gorm:"size:255;not null;uniqueIndex:ux_university_per_region"`
	Description string `json:"description"`

	Region   Region             `json:"region" gorm:"foreignKey:RegionID"`
	Programs []EducationProgram `json:"programs" gorm:"foreignKey:UniversityID"`
}

// EducationProgram — единица выбора в заявке на поступление.
type EducationProgram struct {
	ID           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UniversityID int64  `json:"university_id" gorm:"not null;index"`
	Code         string `json:"code" gorm:"size:32"`
	Name         string `json:"name" gorm:"size:255;not null"`
	Description  string `json:"description"`

	University       University              `json:"university" gorm:"foreignKey:UniversityID"`
	ExamCombinations []ExamCombination       `json:"exam_combinations" gorm:"foreignKey:EducationProgramID"`
	AdmissionScores  []AdmissionScoreHistory `json:"admission_scores" gorm:"foreignKey:EducationProgramID"`
}

// CareerDirectionEducationProgram — явная связь карьерного направления и образовательной программы.
type CareerDirectionEducationProgram struct {
	CareerDirectionID  int64 `json:"career_direction_id" gorm:"primaryKey"`
	EducationProgramID int64 `json:"education_program_id" gorm:"primaryKey"`

	CareerDirection  CareerDirection  `json:"career_direction" gorm:"foreignKey:CareerDirectionID"`
	EducationProgram EducationProgram `json:"education_program" gorm:"foreignKey:EducationProgramID"`
}

// ExamCombination действует для одной программы и одного года приёмной кампании.
type ExamCombination struct {
	ID                 int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	EducationProgramID int64   `json:"education_program_id" gorm:"not null;uniqueIndex:ux_exam_combination"`
	AdmissionYear      int16   `json:"admission_year" gorm:"not null;uniqueIndex:ux_exam_combination;check:admission_year BETWEEN 2020 AND 2100"`
	Name               *string `json:"name,omitempty" gorm:"size:160;uniqueIndex:ux_exam_combination"`

	EducationProgram EducationProgram      `json:"education_program" gorm:"foreignKey:EducationProgramID"`
	Items            []ExamCombinationItem `json:"items" gorm:"foreignKey:ExamCombinationID"`
}

// ExamCombinationItem — один предмет ЕГЭ и его минимальный балл в комбинации.
type ExamCombinationItem struct {
	ExamCombinationID int64  `json:"exam_combination_id" gorm:"primaryKey"`
	ExamSubjectID     int64  `json:"exam_subject_id" gorm:"primaryKey"`
	MinScore          *int16 `json:"min_score" gorm:"check:min_score BETWEEN 0 AND 100"`

	ExamSubject ExamSubject `json:"exam_subject" gorm:"foreignKey:ExamSubjectID"`
}

// AdmissionScoreHistory хранит исторические суммарные проходные баллы для сравнения программ.
type AdmissionScoreHistory struct {
	EducationProgramID int64  `json:"education_program_id" gorm:"primaryKey"`
	AdmissionYear      int16  `json:"admission_year" gorm:"primaryKey;check:admission_year BETWEEN 2020 AND 2100"`
	BudgetPassingScore *int16 `json:"budget_passing_score" gorm:"check:budget_passing_score BETWEEN 0 AND 400"`
	PaidPassingScore   *int16 `json:"paid_passing_score" gorm:"check:paid_passing_score BETWEEN 0 AND 400"`
}

// AdmissionCampaignRule делает лимиты подачи заявлений настраиваемыми по годам.
type AdmissionCampaignRule struct {
	AdmissionYear            int16 `json:"admission_year" gorm:"primaryKey;check:admission_year BETWEEN 2020 AND 2100"`
	MaxUniversities          int16 `json:"max_universities" gorm:"not null;check:max_universities > 0"`
	MaxProgramsPerUniversity int16 `json:"max_programs_per_university" gorm:"not null;check:max_programs_per_university > 0"`
}
