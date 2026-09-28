package employerapi

import "time"

type AccountResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Role      string `json:"role"`
	CompanyID *int64 `json:"company_id,omitempty"`
	IsActive  bool   `json:"is_active"`
	Token     string `json:"token,omitempty"`
}
type CreateAccountInput struct {
	Login     string `json:"login"`
	Role      string `json:"role"`
	CompanyID *int64 `json:"company_id,omitempty"`
}
type UpdateAccountInput struct {
	Login     *string `json:"login,omitempty"`
	CompanyID *int64  `json:"company_id,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

type CompanyInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
}
type CompanyResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
	IsActive    bool   `json:"is_active"`
}

type CareerDirectionInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type TagWeight struct {
	InterestTagID int64   `json:"interest_tag_id"`
	Weight        float64 `json:"weight"`
}
type CareerDirectionResponse struct {
	ID                  int64       `json:"id"`
	CompanyID           int64       `json:"company_id"`
	Name                string      `json:"name"`
	Description         string      `json:"description"`
	IsActive            bool        `json:"is_active"`
	InterestTags        []TagWeight `json:"interest_tags"`
	EducationProgramIDs []int64     `json:"education_program_ids"`
}
type DirectionProgramsInput struct {
	EducationProgramIDs []int64 `json:"education_program_ids"`
}
type DirectionTagsInput struct {
	InterestTags []TagWeight `json:"interest_tags"`
}

type OpportunityInput struct {
	CareerDirectionID int64  `json:"career_direction_id"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	URL               string `json:"url"`
	MinStudyYear      int16  `json:"min_study_year"`
	RegionID          *int64 `json:"region_id,omitempty"`
	IsActive          bool   `json:"is_active"`
}
type OpportunityResponse struct {
	ID int64 `json:"id"`
	OpportunityInput
	UpdatedAt time.Time `json:"updated_at"`
}

type UniversityInput struct {
	RegionID    int64  `json:"region_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
}
type UniversityResponse struct {
	ID int64 `json:"id"`
	UniversityInput
}
type EducationProgramInput struct {
	UniversityID int64  `json:"university_id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Description  string `json:"description"`
}
type EducationProgramResponse struct {
	ID int64 `json:"id"`
	EducationProgramInput
}
type ExamCombinationItemInput struct {
	ExamSubjectID int64  `json:"exam_subject_id"`
	MinScore      *int16 `json:"min_score,omitempty"`
}
type ExamCombinationInput struct {
	EducationProgramID int64                      `json:"education_program_id"`
	AdmissionYear      int16                      `json:"admission_year"`
	Name               *string                    `json:"name,omitempty"`
	Items              []ExamCombinationItemInput `json:"items"`
}
type ExamCombinationResponse struct {
	ID int64 `json:"id"`
	ExamCombinationInput
}
type AdmissionScoreInput struct {
	EducationProgramID int64  `json:"education_program_id"`
	AdmissionYear      int16  `json:"admission_year"`
	BudgetPassingScore *int16 `json:"budget_passing_score,omitempty"`
	PaidPassingScore   *int16 `json:"paid_passing_score,omitempty"`
}
type AdmissionRuleInput struct {
	AdmissionYear            int16 `json:"admission_year"`
	MaxUniversities          int16 `json:"max_universities"`
	MaxProgramsPerUniversity int16 `json:"max_programs_per_university"`
}
type RoadmapTemplateInput struct {
	CareerDirectionID int64              `json:"career_direction_id"`
	Name              string             `json:"name"`
	Steps             []RoadmapStepInput `json:"steps"`
}
type RoadmapStepInput struct {
	StepType    string `json:"step_type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type ReferenceDataResponse struct {
	Regions      []NamedReference `json:"regions"`
	InterestTags []NamedReference `json:"interest_tags"`
	ExamSubjects []NamedReference `json:"exam_subjects"`
}
type NamedReference struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type EmployerApplicationListItem struct {
	RoadmapID       int64                  `json:"application_id"`
	UserID          int64                  `json:"user_id"`
	Grade           int16                  `json:"grade"`
	UserRegion      string                 `json:"user_region"`
	CareerDirection string                 `json:"career_direction"`
	OpportunityID   int64                  `json:"opportunity_id"`
	OpportunityName string                 `json:"opportunity_name"`
	SubmittedAt     time.Time              `json:"submitted_at"`
	Status          string                 `json:"status"`
	Message         string                 `json:"message,omitempty"`
	Contact         string                 `json:"contact,omitempty"`
	FeedbackHistory []EmployerFeedbackView `json:"feedback_history"`
}
type EmployerFeedbackView struct {
	ID        int64     `json:"id"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Contact   string    `json:"contact,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
type EmployerFeedbackInput struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Contact string `json:"contact,omitempty"`
}
