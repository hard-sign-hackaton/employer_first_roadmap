package models

type UserState string

const (
	UserStateStart           UserState = "start"
	UserStateSurveyCompleted UserState = "survey_completed"

	// Small survey
	UserStateSmallSurveyWaitingCompanyName      UserState = "small_survey_waiting_company_name"
	UserStateSmallSurveyWaitingGrade            UserState = "small_survey_waiting_grade"
	UserStateSmallSurveyWaitingRegion           UserState = "small_survey_waiting_region"
	UserStateSmallSurveyWaitingRelocation       UserState = "small_survey_waiting_relocation"
	UserStateSmallSurveyWaitingExamSelection    UserState = "small_survey_waiting_exam_selection"
	UserStateSmallSurveyWaitingExamSubject      UserState = "small_survey_waiting_exam_subject"
	UserStateSmallSurveyWaitingExamScore        UserState = "small_survey_waiting_exam_score"
	UserStateSmallSurveyWaitingDirection        UserState = "small_survey_waiting_direction"
	UserStateSmallSurveyWaitingExamSet          UserState = "small_survey_waiting_exam_set"
	UserStateSmallSurveyWaitingGoalConfirmation UserState = "small_survey_waiting_goal_confirmation"

	// Big survey
	UserStateBigSurveyWaitingGrade            UserState = "big_survey_waiting_grade"
	UserStateBigSurveyWaitingRegion           UserState = "big_survey_waiting_region"
	UserStateBigSurveyWaitingRelocation       UserState = "big_survey_waiting_relocation"
	UserStateBigSurveyWaitingExamSelection    UserState = "big_survey_waiting_exam_selection"
	UserStateBigSurveyWaitingExamSubject      UserState = "big_survey_waiting_exam_subject"
	UserStateBigSurveyWaitingExamScore        UserState = "big_survey_waiting_exam_score"
	UserStateBigSurveyWaitingInterest         UserState = "big_survey_waiting_interest"
	UserStateBigSurveyWaitingSubjects         UserState = "big_survey_waiting_subjects"
	UserStateBigSurveyWaitingCompany          UserState = "big_survey_waiting_company"
	UserStateBigSurveyWaitingDirection        UserState = "big_survey_waiting_direction"
	UserStateBigSurveyWaitingExamSet          UserState = "big_survey_waiting_exam_set"
	UserStateBigSurveyWaitingGoalConfirmation UserState = "big_survey_waiting_goal_confirmation"

	// Roadmap: сценарий развития пользователя после опроса
	UserStateRoadmapExamChoice         UserState = "roadmap_exam_choice"
	UserStateRoadmapExamChoiceSubjects UserState = "roadmap_exam_choice_subjects"
	UserStateRoadmapExamScores         UserState = "roadmap_exam_scores"
	UserStateRoadmapUniversityOptions  UserState = "roadmap_university_options"
	UserStateRoadmapUniversityConfirm  UserState = "roadmap_university_confirm"
	UserStateRoadmapPostAdmission      UserState = "roadmap_post_admission"
)
