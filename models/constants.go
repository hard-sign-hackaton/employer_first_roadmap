package models

const (
	GoalStatusActive    = "active"
	GoalStatusArchived  = "archived"
	GoalStatusCompleted = "completed"
)

const (
	RoadmapStatusActive    = "active"
	RoadmapStatusArchived  = "archived"
	RoadmapStatusCompleted = "completed"
)

const (
	SubjectStatusPlanned  = "planned"
	SubjectStatusSelected = "selected"
	SubjectStatusPassed   = "passed"
)

const (
	RoadmapStepStatusPending   = "pending"
	RoadmapStepStatusActive    = "active"
	RoadmapStepStatusCompleted = "completed"
	RoadmapStepStatusSkipped   = "skipped"
)

const (
	RoadmapStepTypeChooseOrConfirmExams     = "choose_or_confirm_exams"
	RoadmapStepTypePrepareForExams          = "prepare_for_exams"
	RoadmapStepTypePassExams                = "pass_exams"
	RoadmapStepTypeChooseUniversity         = "choose_university"
	RoadmapStepTypeSubmitAdmissionDocuments = "submit_admission_documents"
	RoadmapStepTypeConfirmEnrollment        = "confirm_enrollment"
	RoadmapStepTypeLearnAtUniversity        = "learn_at_university"
	RoadmapStepTypeEmployerExperience       = "employer_experience"
	RoadmapStepTypeApplyToEmployer          = "apply_to_employer"
)

const (
	AdmissionApplicationStatusPlanned   = "planned"
	AdmissionApplicationStatusSubmitted = "submitted"
	AdmissionApplicationStatusWithdrawn = "withdrawn"
)

const (
	EnrollmentStatusChosen      = "chosen"
	EnrollmentStatusNotEnrolled = "not_enrolled"
)

const (
	OpportunityTypeInternship       = "internship"
	OpportunityTypePractice         = "practice"
	OpportunityTypeProject          = "project"
	OpportunityTypeHackathon        = "hackathon"
	OpportunityTypeTargetedTraining = "targeted_training"
)
