package dto

// EducationOptionResponse — вариант поступления, рассчитанный после получения результатов ЕГЭ.
type EducationOptionResponse struct {
	UniversityID       int64    `json:"university_id"`
	UniversityName     string   `json:"university_name"`
	UniversityRegion   string   `json:"university_region"`
	EducationProgramID int64    `json:"education_program_id"`
	ProgramCode        string   `json:"program_code,omitempty"`
	ProgramName        string   `json:"program_name"`
	AdmissionYear      int16    `json:"admission_year"`
	RequiredSubjects   []string `json:"required_subjects"`
	MinimumTotalScore  *int16   `json:"minimum_total_score,omitempty"`
	BudgetPassingScore *int16   `json:"budget_passing_score,omitempty"`
	PaidPassingScore   *int16   `json:"paid_passing_score,omitempty"`
	Explanation        []string `json:"explanation"`
}

// FindEducationOptionsRequest запускает подбор ОП и вузов после результатов ЕГЭ.
// ExpandGeography=true означает повторный поиск без исходного ограничения по региону.
type FindEducationOptionsRequest struct {
	RoadmapID       int64 `json:"roadmap_id"`
	ExpandGeography bool  `json:"expand_geography"`
}

// AdmissionPlanLimitsResponse сообщает актуальные для кампании лимиты плана подачи.
type AdmissionPlanLimitsResponse struct {
	AdmissionYear            int16 `json:"admission_year"`
	MaxUniversities          int16 `json:"max_universities"`
	MaxProgramsPerUniversity int16 `json:"max_programs_per_university"`
}

// GetAdmissionPlanRequest запрашивает сохранённый план подачи документов для roadmap.
type GetAdmissionPlanRequest struct {
	RoadmapID int64 `json:"roadmap_id"`
}

// SaveAdmissionPlanRequest сохраняет выбранные после ЕГЭ варианты подачи документов.
// Сервис проверяет ограничения кампании по числу вузов и программ в каждом вузе.
type SaveAdmissionPlanRequest struct {
	Applications []AdmissionPlanItemInput `json:"applications"`
}

// AdmissionPlanItemInput — одна программа в плане подачи документов.
type AdmissionPlanItemInput struct {
	EducationProgramID int64 `json:"education_program_id"`
}

// AdmissionApplicationResponse — сохранённый пункт плана подачи.
type AdmissionApplicationResponse struct {
	ID                 int64  `json:"id"`
	UniversityID       int64  `json:"university_id"`
	UniversityName     string `json:"university_name"`
	EducationProgramID int64  `json:"education_program_id"`
	ProgramName        string `json:"program_name"`
	Status             string `json:"status"`
}

// SaveEnrollmentChoiceRequest фиксирует единственный итог приёмной кампании.
// При status=chosen обязателен admission_application_id; при not_enrolled он не передаётся.
type SaveEnrollmentChoiceRequest struct {
	Status                 string `json:"status"`
	AdmissionApplicationID *int64 `json:"admission_application_id,omitempty"`
	EnrollmentYear         *int16 `json:"enrollment_year,omitempty"`
}

// EnrollmentChoiceResponse — итоговое решение о поступлении.
type EnrollmentChoiceResponse struct {
	Status                 string `json:"status"`
	AdmissionApplicationID *int64 `json:"admission_application_id,omitempty"`
	UniversityName         string `json:"university_name,omitempty"`
	EducationProgramName   string `json:"education_program_name,omitempty"`
	EnrollmentYear         *int16 `json:"enrollment_year,omitempty"`
}
