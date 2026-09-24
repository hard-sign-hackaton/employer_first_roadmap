package utils

import (
	"efr_bot/models"
	"sync"
)

var (
	UserStateStorage   = make(map[int64]models.UserState)
	SmallSurveyStorage = make(map[int64]*SmallSurveyData)
	mu                 sync.Mutex
)

func UpdateUserStateStorage(id int64, newState models.UserState) {
	mu.Lock()
	UserStateStorage[id] = newState
	mu.Unlock()
}

type SmallSurveyData struct {
	Grade                                  int16
	RegionID, CompanyID, CareerDirectionID int64
	PlannedExamIDs                         []int64
	CompanyName, CareerDirectionName       string
	PlannedExamNames                       []string
	SelectedExamIDs                        []int64
	SelectedExamNames                      []string
	SelectedExamScores                     map[int64]*int16
	ExamScoreStep                          int
	WillingToRelocate                      bool
	CompanyIDs, RegionIDs, DirectionIDs    []int64
	DirectionNames                         []string
	ExamSets                               [][]int64
	ExamSetNames                           [][]string
	AvailableExamIDs                       []int64
	AvailableExamNames                     []string
	ActivityTagIDs                         []int64
	ActivityTagNames                       []string
	SelectedActivityTagIDs                 []int64
	SelectedSchoolSubjectIDs               []int64
	SelectedSchoolSubjectNames             []string
}

func ResetSmallSurvey(id int64) { mu.Lock(); SmallSurveyStorage[id] = &SmallSurveyData{}; mu.Unlock() }
func GetSmallSurvey(id int64) *SmallSurveyData {
	mu.Lock()
	defer mu.Unlock()
	if SmallSurveyStorage[id] == nil {
		SmallSurveyStorage[id] = &SmallSurveyData{}
	}
	return SmallSurveyStorage[id]
}

func GetUserState(id int64) models.UserState {
	mu.Lock()
	state := UserStateStorage[id]
	mu.Unlock()

	return state
}
