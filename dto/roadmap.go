package dto

import "time"

// RoadmapStepResponse — персональный шаг roadmap, который нужно показать пользователю.
type RoadmapStepResponse struct {
	ID              int64      `json:"id"`
	OrderNo         int16      `json:"order_no"`
	StepType        string     `json:"step_type"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	ActionURL       string     `json:"action_url,omitempty"`
	TargetStudyYear *int16     `json:"target_study_year,omitempty"`
	Status          string     `json:"status"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// RoadmapResponse — текущий снимок roadmap и следующее действие пользователя.
type RoadmapResponse struct {
	ID                  int64                        `json:"id"`
	GoalID              int64                        `json:"goal_id"`
	Goal                RoadmapGoalResponse          `json:"goal"`
	Status              string                       `json:"status"`
	Steps               []RoadmapStepResponse        `json:"steps"`
	NextAction          *RoadmapStepResponse         `json:"next_action,omitempty"`
	EnrollmentChoice    *EnrollmentChoiceResponse    `json:"enrollment_choice,omitempty"`
	EmployerApplication *EmployerApplicationResponse `json:"employer_application,omitempty"`
}

// RoadmapGoalResponse — краткая цель, которая показывается при возвращении в roadmap.
type RoadmapGoalResponse struct {
	CompanyName         string `json:"company_name"`
	CareerDirectionName string `json:"career_direction_name"`
	TargetAdmissionYear int16  `json:"target_admission_year"`
}

// CreateRoadmapRequest создаёт общий roadmap для уже подтверждённой цели.
type CreateRoadmapRequest struct {
	GoalID int64 `json:"goal_id"`
}

// GetRoadmapRequest запрашивает roadmap по его идентификатору.
type GetRoadmapRequest struct {
	RoadmapID int64 `json:"roadmap_id"`
}

// GetRoadmapStepRequest запрашивает конкретный шаг roadmap.
type GetRoadmapStepRequest struct {
	RoadmapID int64 `json:"roadmap_id"`
	StepID    int64 `json:"step_id"`
}

// UpdateRoadmapStepRequest меняет статус шага. Для completed сервис фиксирует время завершения.
type UpdateRoadmapStepRequest struct {
	Status string `json:"status"`
}

// ReconsiderTrajectoryRequest запускает единое действие «Пересмотреть траекторию».
// Причина нужна сервису для корректного следующего сценария, например пересдачи ЕГЭ.
type ReconsiderTrajectoryRequest struct {
	Reason string `json:"reason"`
}

// CompanyOpportunityResponse — конкретный шаг практического опыта у выбранного работодателя.
type CompanyOpportunityResponse struct {
	ID                int64  `json:"id"`
	CompanyName       string `json:"company_name"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	URL               string `json:"url,omitempty"`
	CompanyWebsiteURL string `json:"company_website_url,omitempty"`
	MinStudyYear      int16  `json:"min_study_year"`
	IsAvailable       bool   `json:"is_available"`
}

// EmployerApplicationResponse подтверждает отправку заявки работодателю.
type EmployerApplicationResponse struct {
	RoadmapID            int64                      `json:"roadmap_id"`
	CompanyOpportunityID int64                      `json:"company_opportunity_id"`
	CompanyName          string                     `json:"company_name,omitempty"`
	OpportunityName      string                     `json:"opportunity_name,omitempty"`
	Status               string                     `json:"status"`
	Message              string                     `json:"message,omitempty"`
	Contact              string                     `json:"contact,omitempty"`
	UpdatedAt            *time.Time                 `json:"updated_at,omitempty"`
	History              []EmployerFeedbackResponse `json:"history,omitempty"`
}

type EmployerFeedbackResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Contact   string    `json:"contact,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
