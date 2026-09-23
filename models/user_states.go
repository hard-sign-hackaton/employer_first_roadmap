package models

type UserState string

const (
	UserStateStart           UserState = "start"
	UserStateSurveyCompleted UserState = "survey_completed"

	// Small survey
	UserStateSmallSurveyWaitingCompanyName   UserState = "small_survey_waiting_company_name"
	UserStateSmallSurveyWaitingGrade         UserState = "small_survey_waiting_grade"
	UserStateSmallSurveyWaitingRegion        UserState = "small_survey_waiting_region"
	UserStateSmallSurveyWaitingRelocation    UserState = "small_survey_waiting_relocation"
	UserStateSmallSurveyWaitingExamSelection UserState = "small_survey_waiting_exam_selection"

	// Big survey
	UserStateBigSurveyWaitingGrade         UserState = "big_survey_waiting_grade"
	UserStateBigSurveyWaitingRegion        UserState = "big_survey_waiting_region"
	UserStateBigSurveyWaitingRelocation    UserState = "big_survey_waiting_relocation"
	UserStateBigSurveyWaitingExamSelection UserState = "big_survey_waiting_exam_selection"
	UserStateBigSurveyWaitingExamScore     UserState = "big_survey_waiting_exam_score"
	UserStateBigSurveyWaitingInterest      UserState = "big_survey_waiting_interest"
	UserStateBigSurveyWaitingSubjects      UserState = "big_survey_waiting_subjects"
	UserStateBigSurveyWaitingCompany       UserState = "big_survey_waiting_company"
)
