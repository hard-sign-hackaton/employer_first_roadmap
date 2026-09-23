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
	ID         int64                 `json:"id"`
	GoalID     int64                 `json:"goal_id"`
	Status     string                `json:"status"`
	Steps      []RoadmapStepResponse `json:"steps"`
	NextAction *RoadmapStepResponse  `json:"next_action,omitempty"`
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
	ID           int64  `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	URL          string `json:"url,omitempty"`
	MinStudyYear int16  `json:"min_study_year"`
	IsAvailable  bool   `json:"is_available"`
}
