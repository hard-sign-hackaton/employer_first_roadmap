package models

type UserState string

const (
	UserStateStart           UserState = "start"
	UserStateSurveyCompleted UserState = "survey_completed"

	// Small survey
	UserStateSmallSurveyWaitingCompanyName UserState = "small_survey_waiting_company_name"
	UserStateSmallSurveyWaitingGrade       UserState = "small_survey_waiting_grade"
	UserStateSmallSurveyWaitingRegion      UserState = "small_survey_waiting_region"
	UserStateSmallSurveyWaitingRelocation  UserState = "small_survey_waiting_relocation"

	// Big survey
)
