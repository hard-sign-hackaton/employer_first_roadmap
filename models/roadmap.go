package models

import "time"

// UserGoal создаётся после подтверждения компании, карьерного направления и набора ЕГЭ.
// Вуз и ОП выбираются позже, после получения фактических результатов ЕГЭ.
type UserGoal struct {
	ID                  int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserProfileID       int64      `json:"user_profile_id" gorm:"not null;index"`
	CareerDirectionID   int64      `json:"career_direction_id" gorm:"not null;index"`
	TargetAdmissionYear int16      `json:"target_admission_year" gorm:"not null;check:target_admission_year BETWEEN 2020 AND 2100"`
	Status              string     `json:"status" gorm:"size:16;not null;check:status IN ('active','archived','completed')"`
	CreatedAt           time.Time  `json:"created_at"`
	ArchivedAt          *time.Time `json:"archived_at"`

	CareerDirection CareerDirection `json:"career_direction" gorm:"foreignKey:CareerDirectionID"`
	Roadmaps        []Roadmap       `json:"roadmaps" gorm:"foreignKey:UserGoalID"`
}

// RoadmapTemplate — переиспользуемый общий шаблон roadmap для одного карьерного направления.
type RoadmapTemplate struct {
	ID                int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	CareerDirectionID int64  `json:"career_direction_id" gorm:"not null;uniqueIndex:ux_roadmap_template_version"`
	Name              string `json:"name" gorm:"size:200;not null"`
	Version           int    `json:"version" gorm:"not null;uniqueIndex:ux_roadmap_template_version;check:version > 0"`
	IsActive          bool   `json:"is_active" gorm:"not null;default:true"`

	CareerDirection CareerDirection       `json:"career_direction" gorm:"foreignKey:CareerDirectionID"`
	Steps           []RoadmapTemplateStep `json:"steps" gorm:"foreignKey:RoadmapTemplateID"`
}

// RoadmapTemplateStep копируется в RoadmapStep при создании roadmap.
type RoadmapTemplateStep struct {
	ID                int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	RoadmapTemplateID int64  `json:"roadmap_template_id" gorm:"not null;uniqueIndex:ux_template_step_order"`
	OrderNo           int16  `json:"order_no" gorm:"not null;uniqueIndex:ux_template_step_order;check:order_no > 0"`
	StepType          string `json:"step_type" gorm:"size:40;not null"`
	Title             string `json:"title" gorm:"size:255;not null"`
	Description       string `json:"description"`
}

// Roadmap — активный, архивный или завершённый снимок цели пользователя.
type Roadmap struct {
	ID                int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserGoalID        int64      `json:"user_goal_id" gorm:"not null;index"`
	RoadmapTemplateID int64      `json:"roadmap_template_id" gorm:"not null"`
	Status            string     `json:"status" gorm:"size:16;not null;check:status IN ('active','archived','completed')"`
	CreatedAt         time.Time  `json:"created_at"`
	ArchivedAt        *time.Time `json:"archived_at"`
	CompletedAt       *time.Time `json:"completed_at"`

	UserGoal              UserGoal                      `json:"user_goal" gorm:"foreignKey:UserGoalID"`
	RoadmapTemplate       RoadmapTemplate               `json:"roadmap_template" gorm:"foreignKey:RoadmapTemplateID"`
	Steps                 []RoadmapStep                 `json:"steps" gorm:"foreignKey:RoadmapID"`
	AdmissionApplications []RoadmapAdmissionApplication `json:"admission_applications" gorm:"foreignKey:RoadmapID"`
	EnrollmentChoice      *RoadmapEnrollmentChoice      `json:"enrollment_choice" gorm:"foreignKey:RoadmapID"`
	EmployerApplication   *RoadmapEmployerApplication   `json:"employer_application" gorm:"foreignKey:RoadmapID"`
}

// RoadmapStep — персональный снимок шага. Незавершённые шаги могут уточняться позднее.
type RoadmapStep struct {
	ID                   int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	RoadmapID            int64      `json:"roadmap_id" gorm:"not null;uniqueIndex:ux_roadmap_step_order"`
	TemplateStepID       *int64     `json:"template_step_id"`
	CompanyOpportunityID *int64     `json:"company_opportunity_id"`
	OrderNo              int16      `json:"order_no" gorm:"not null;uniqueIndex:ux_roadmap_step_order;check:order_no > 0"`
	StepType             string     `json:"step_type" gorm:"size:40;not null"`
	Title                string     `json:"title" gorm:"size:255;not null"`
	Description          string     `json:"description"`
	ActionURL            string     `json:"action_url"`
	TargetStudyYear      *int16     `json:"target_study_year" gorm:"check:target_study_year BETWEEN 1 AND 6"`
	Status               string     `json:"status" gorm:"size:16;not null;check:status IN ('pending','active','completed','skipped')"`
	CompletedAt          *time.Time `json:"completed_at"`

	TemplateStep       *RoadmapTemplateStep `json:"template_step,omitempty" gorm:"foreignKey:TemplateStepID"`
	CompanyOpportunity *CompanyOpportunity  `json:"company_opportunity,omitempty" gorm:"foreignKey:CompanyOpportunityID"`
}

// RoadmapAdmissionApplication — одна программа в плане подачи пользователя после ЕГЭ.
type RoadmapAdmissionApplication struct {
	ID                 int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	RoadmapID          int64     `json:"roadmap_id" gorm:"not null;uniqueIndex:ux_roadmap_application"`
	EducationProgramID int64     `json:"education_program_id" gorm:"not null;uniqueIndex:ux_roadmap_application"`
	Status             string    `json:"status" gorm:"size:16;not null;check:status IN ('planned','submitted','withdrawn')"`
	CreatedAt          time.Time `json:"created_at"`

	EducationProgram EducationProgram `json:"education_program" gorm:"foreignKey:EducationProgramID"`
}

// RoadmapEnrollmentChoice хранит только итоговый выбор пользователя, а не результаты по каждой заявке.
type RoadmapEnrollmentChoice struct {
	RoadmapID              int64     `json:"roadmap_id" gorm:"primaryKey"`
	AdmissionApplicationID *int64    `json:"admission_application_id"`
	Status                 string    `json:"status" gorm:"size:16;not null;check:status IN ('chosen','not_enrolled')"`
	EnrollmentYear         *int16    `json:"enrollment_year" gorm:"check:enrollment_year BETWEEN 2020 AND 2100"`
	CurrentStudyYear       int16     `json:"current_study_year" gorm:"not null;default:1;check:current_study_year BETWEEN 1 AND 6"`
	DecidedAt              time.Time `json:"decided_at"`

	AdmissionApplication *RoadmapAdmissionApplication `json:"admission_application,omitempty" gorm:"foreignKey:AdmissionApplicationID"`
}

// RoadmapEmployerApplication — факт подачи заявки на выбранную возможность работодателя.
type RoadmapEmployerApplication struct {
	RoadmapID            int64     `json:"roadmap_id" gorm:"primaryKey"`
	CompanyOpportunityID int64     `json:"company_opportunity_id" gorm:"not null"`
	Status               string    `json:"status" gorm:"size:16;not null;check:status IN ('submitted')"`
	SubmittedAt          time.Time `json:"submitted_at"`

	CompanyOpportunity CompanyOpportunity `json:"company_opportunity" gorm:"foreignKey:CompanyOpportunityID"`
}
