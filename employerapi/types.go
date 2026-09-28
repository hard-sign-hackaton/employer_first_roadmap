package employerapi

import "time"

// CatalogRequest is the complete employer-owned data set consumed by the bot.
// Names are used for reference entities so an employer UI does not have to know
// database identifiers in advance.
type CatalogRequest struct {
	Company       CompanyInput           `json:"company"`
	Directions    []CareerDirectionInput `json:"directions"`
	AdmissionRule *AdmissionRuleInput    `json:"admission_rule,omitempty"`
}

type CompanyInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
}

type CareerDirectionInput struct {
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	InterestTags      []InterestTagInput      `json:"interest_tags"`
	EducationPrograms []EducationProgramInput `json:"education_programs"`
	Opportunities     []OpportunityInput      `json:"opportunities"`
	Roadmap           *RoadmapTemplateInput   `json:"roadmap,omitempty"`
}

type InterestTagInput struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type EducationProgramInput struct {
	Code             string                 `json:"code"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	University       UniversityInput        `json:"university"`
	ExamCombinations []ExamCombinationInput `json:"exam_combinations"`
	AdmissionScores  []AdmissionScoreInput  `json:"admission_scores,omitempty"`
}

type UniversityInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
	Region      string `json:"region"`
}

type ExamCombinationInput struct {
	AdmissionYear int16              `json:"admission_year"`
	Name          string             `json:"name"`
	Subjects      []ExamSubjectInput `json:"subjects"`
}

type ExamSubjectInput struct {
	Name     string `json:"name"`
	MinScore *int16 `json:"min_score,omitempty"`
}

type AdmissionScoreInput struct {
	AdmissionYear      int16  `json:"admission_year"`
	BudgetPassingScore *int16 `json:"budget_passing_score,omitempty"`
	PaidPassingScore   *int16 `json:"paid_passing_score,omitempty"`
}

type OpportunityInput struct {
	ID           int64  `json:"id,omitempty"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	URL          string `json:"url"`
	MinStudyYear int16  `json:"min_study_year"`
	Region       string `json:"region,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

type RoadmapTemplateInput struct {
	Name  string             `json:"name"`
	Steps []RoadmapStepInput `json:"steps"`
}

type RoadmapStepInput struct {
	StepType    string `json:"step_type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AdmissionRuleInput struct {
	AdmissionYear            int16 `json:"admission_year"`
	MaxUniversities          int16 `json:"max_universities"`
	MaxProgramsPerUniversity int16 `json:"max_programs_per_university"`
}

type CatalogResponse struct {
	Company       CompanyResponse           `json:"company"`
	Directions    []CareerDirectionResponse `json:"directions"`
	AdmissionRule *AdmissionRuleInput       `json:"admission_rule,omitempty"`
}

type CompanyResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
}

type CareerDirectionResponse struct {
	ID                int64                      `json:"id"`
	Name              string                     `json:"name"`
	Description       string                     `json:"description"`
	InterestTags      []InterestTagResponse      `json:"interest_tags"`
	EducationPrograms []EducationProgramResponse `json:"education_programs"`
	Opportunities     []OpportunityResponse      `json:"opportunities"`
	Roadmap           *RoadmapTemplateResponse   `json:"roadmap,omitempty"`
}

type InterestTagResponse struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type EducationProgramResponse struct {
	ID               int64                 `json:"id"`
	Code             string                `json:"code"`
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	University       UniversityResponse    `json:"university"`
	ExamCombinations []ExamCombinationView `json:"exam_combinations"`
	AdmissionScores  []AdmissionScoreInput `json:"admission_scores"`
}

type UniversityResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WebsiteURL  string `json:"website_url"`
	RegionID    int64  `json:"region_id"`
	Region      string `json:"region"`
}

type ExamCombinationView struct {
	ID            int64             `json:"id"`
	AdmissionYear int16             `json:"admission_year"`
	Name          string            `json:"name"`
	Subjects      []ExamSubjectView `json:"subjects"`
}

type ExamSubjectView struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	MinScore *int16 `json:"min_score,omitempty"`
}

type OpportunityResponse struct {
	ID           int64     `json:"id"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	URL          string    `json:"url"`
	MinStudyYear int16     `json:"min_study_year"`
	RegionID     *int64    `json:"region_id,omitempty"`
	Region       string    `json:"region,omitempty"`
	IsActive     bool      `json:"is_active"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RoadmapTemplateResponse struct {
	ID       int64              `json:"id"`
	Name     string             `json:"name"`
	Version  int                `json:"version"`
	IsActive bool               `json:"is_active"`
	Steps    []RoadmapStepInput `json:"steps"`
}

type ReferenceDataResponse struct {
	Regions          []NamedReference `json:"regions"`
	InterestTags     []NamedReference `json:"interest_tags"`
	ExamSubjects     []NamedReference `json:"exam_subjects"`
	OpportunityTypes []string         `json:"opportunity_types"`
	RoadmapStepTypes []string         `json:"roadmap_step_types"`
}

type NamedReference struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type opportunityStatusRequest struct {
	IsActive bool `json:"is_active"`
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
