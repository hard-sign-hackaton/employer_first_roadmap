package dto

// CompanyCatalogItemResponse — краткая карточка работодателя для каталога и рекомендаций.
type CompanyCatalogItemResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// FindCompaniesRequest задаёт поиск работодателя в каталоге для минимального опроса.
type FindCompaniesRequest struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// RecommendCompaniesRequest запускает рекомендации работодателей после полного опроса.
// Для ученика 11 класса сервис дополнительно учитывает уже выбранные ЕГЭ из профиля.
type RecommendCompaniesRequest struct {
	Limit int `json:"limit,omitempty"`
}

// RecommendedCompanyResponse — карточка работодателя с объяснением причины рекомендации.
type RecommendedCompanyResponse struct {
	CompanyCatalogItemResponse
	Score        float64  `json:"score"`
	Reasons      []string `json:"reasons"`
	IsCompatible bool     `json:"is_compatible_with_selected_exams"`
}

// SelectCompanyRequest подтверждает работодателя, найденного в каталоге или рекомендациях.
type SelectCompanyRequest struct {
	CompanyID int64 `json:"company_id"`
}

// GetCareerDirectionsRequest запрашивает направления подтверждённого работодателя.
// Для ученика 11 класса сервис учитывает выбранные ЕГЭ из его профиля.
type GetCareerDirectionsRequest struct {
	CompanyID int64 `json:"company_id"`
}

// CareerDirectionResponse — карьерное направление выбранного работодателя.
type CareerDirectionResponse struct {
	ID          int64    `json:"id"`
	CompanyID   int64    `json:"company_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Score       *float64 `json:"score,omitempty"`
	Reasons     []string `json:"reasons,omitempty"`
}

// RecommendedExamSetResponse — рекомендованный набор ЕГЭ для направления.
// Связанные ОП и вузы используются сервисом внутри и не показываются на этом шаге пользователю.
type RecommendedExamSetResponse struct {
	ExamSubjectIDs []int64               `json:"exam_subject_ids"`
	Subjects       []ExamSubjectResponse `json:"subjects"`
	Description    string                `json:"description,omitempty"`
	SourceYear     int16                 `json:"source_year"`
}

// GetRecommendedExamSetsRequest запрашивает наборы ЕГЭ для выбранного карьерного направления.
type GetRecommendedExamSetsRequest struct {
	CareerDirectionID int64 `json:"career_direction_id"`
}

// ConfirmGoalRequest создаёт цель после выбора компании, направления и набора ЕГЭ.
type ConfirmGoalRequest struct {
	CareerDirectionID   int64 `json:"career_direction_id"`
	TargetAdmissionYear int16 `json:"target_admission_year"`
}

// GoalResponse — подтверждённая цель пользователя до выбора вуза и ОП.
type GoalResponse struct {
	ID                  int64                      `json:"id"`
	Company             CompanyCatalogItemResponse `json:"company"`
	CareerDirection     CareerDirectionResponse    `json:"career_direction"`
	TargetAdmissionYear int16                      `json:"target_admission_year"`
	Status              string                     `json:"status"`
	Subjects            []UserSubjectResponse      `json:"subjects"`
}
