package dto

// UpsertProfileRequest сохраняет минимальный профиль из минимального или полного опроса.
type UpsertProfileRequest struct {
	Grade             int16 `json:"grade"`
	RegionID          int64 `json:"region_id"`
	WillingToRelocate bool  `json:"willing_to_relocate"`
}

// ProfileResponse возвращает сохранённый минимальный профиль пользователя.
type ProfileResponse struct {
	UserID            int64          `json:"user_id"`
	Grade             int16          `json:"grade"`
	Region            RegionResponse `json:"region"`
	WillingToRelocate bool           `json:"willing_to_relocate"`
}

// SaveSurveyInterestsRequest сохраняет нормализованные интересы из полного опроса.
// В минимальном опросе этот запрос не вызывается.
type SaveSurveyInterestsRequest struct {
	Interests []InterestInput `json:"interests"`
}

// InterestInput — один интерес и его вес, вычисленный по ответам полного опроса.
type InterestInput struct {
	InterestTagID int64   `json:"interest_tag_id"`
	Weight        float64 `json:"weight"`
}

// InterestResponse — интерес, сохранённый в профиле пользователя.
type InterestResponse struct {
	InterestTagID int64   `json:"interest_tag_id"`
	Name          string  `json:"name"`
	Weight        float64 `json:"weight"`
}

// SaveUserSubjectsRequest сохраняет планируемые или уже выбранные предметы ЕГЭ.
// Количество предметов намеренно не ограничивается DTO: в сценарии их может быть больше трёх.
type SaveUserSubjectsRequest struct {
	Subjects []UserSubjectInput `json:"subjects"`
}

// UserSubjectInput — предмет ЕГЭ, выбранный или запланированный пользователем.
// Status принимает planned либо selected; actual_score заполняется отдельным запросом после ЕГЭ.
type UserSubjectInput struct {
	ExamSubjectID int64  `json:"exam_subject_id"`
	Status        string `json:"status"`
	ExpectedScore *int16 `json:"expected_score,omitempty"`
}

// SaveExamResultsRequest фиксирует фактические результаты ЕГЭ после завершения шага roadmap.
type SaveExamResultsRequest struct {
	Results []ExamResultInput `json:"results"`
}

// ExamResultInput — фактический балл по одному сданному предмету ЕГЭ.
type ExamResultInput struct {
	ExamSubjectID int64 `json:"exam_subject_id"`
	ActualScore   int16 `json:"actual_score"`
}

// UserSubjectResponse — предмет ЕГЭ и текущее состояние его прохождения.
type UserSubjectResponse struct {
	ExamSubjectID int64  `json:"exam_subject_id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	ExpectedScore *int16 `json:"expected_score,omitempty"`
	ActualScore   *int16 `json:"actual_score,omitempty"`
}
