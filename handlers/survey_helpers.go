package handlers

import (
	"context"
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/utils"
	"fmt"
	"strconv"
	"strings"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func showExamSubjectSelection(ctx maxbot.Context, nextState models.UserState) error {
	userID := ctx.Update().UserID
	s := utils.GetSmallSurvey(userID)
	subjects, err := app.Reference.ListExamSubjects(context.Background())
	if err != nil {
		return ctx.Send("Не удалось получить список предметов ЕГЭ.")
	}
	s.AvailableExamIDs = nil
	s.AvailableExamNames = nil
	kb := model.NewKeyboard()
	for index, subject := range subjects {
		s.AvailableExamIDs = append(s.AvailableExamIDs, subject.ID)
		s.AvailableExamNames = append(s.AvailableExamNames, subject.Name)
		kb.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, subject.Name))
	}
	utils.UpdateUserStateStorage(userID, nextState)
	return ctx.Send("Выберите предметы ЕГЭ номерами через запятую, например: 1,3,5.", maxbot.WithKeyboard(kb))
}

func saveSelectedExamSubjects(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	s := utils.GetSmallSurvey(userID)
	inputs := make([]dto.UserSubjectInput, 0, len(s.SelectedExamIDs))
	for _, subjectID := range s.SelectedExamIDs {
		inputs = append(inputs, dto.UserSubjectInput{
			ExamSubjectID: subjectID,
			Status:        models.SubjectStatusSelected,
			ExpectedScore: s.SelectedExamScores[subjectID],
		})
	}
	if _, err := app.Profile.SaveUserSubjects(context.Background(), userID, dto.SaveUserSubjectsRequest{Subjects: inputs}); err != nil {
		return err
	}
	return nil
}

func chooseExamSubjects(s *utils.SmallSurveyData, text string) bool {
	indexes, ok := multipleChoices(text, len(s.AvailableExamIDs))
	if !ok {
		return false
	}
	s.SelectedExamIDs = make([]int64, 0, len(indexes))
	s.SelectedExamNames = make([]string, 0, len(indexes))
	for _, index := range indexes {
		s.SelectedExamIDs = append(s.SelectedExamIDs, s.AvailableExamIDs[index])
		s.SelectedExamNames = append(s.SelectedExamNames, s.AvailableExamNames[index])
	}
	s.SelectedExamScores = make(map[int64]*int16)
	return true
}

func saveExpectedScores(s *utils.SmallSurveyData, text string) bool {
	if strings.EqualFold(strings.TrimSpace(text), "Пропустить") {
		return true
	}
	for _, pair := range strings.Split(text, ",") {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return false
		}
		index, indexErr := strconv.Atoi(strings.TrimSpace(parts[0]))
		score, scoreErr := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 16)
		if indexErr != nil || scoreErr != nil || index < 1 || index > len(s.SelectedExamIDs) || score < 0 || score > 100 {
			return false
		}
		value := int16(score)
		s.SelectedExamScores[s.SelectedExamIDs[index-1]] = &value
	}
	return true
}

func showGoalConfirmation(ctx maxbot.Context, state models.UserState) error {
	userID := ctx.Update().UserID
	s := utils.GetSmallSurvey(userID)
	examNames := s.PlannedExamNames
	if len(s.SelectedExamNames) > 0 {
		examNames = s.SelectedExamNames
	}
	utils.UpdateUserStateStorage(userID, state)
	keyboard := model.NewKeyboard()
	keyboard.AddRow().AddMessage("Подтвердить").AddMessage("Изменить направление")
	return ctx.Send(
		"Цель:\n"+
			"Компания: "+s.CompanyName+"\n"+
			"Направление: "+s.CareerDirectionName+"\n"+
			"ЕГЭ: "+strings.Join(examNames, ", ")+"\n\n"+
			"Подтвердить цель?",
		maxbot.WithKeyboard(keyboard),
	)
}

func multipleChoices(text string, length int) ([]int, bool) {
	seen := make(map[int]struct{})
	result := make([]int, 0)
	for _, part := range strings.Split(text, ",") {
		value, err := strconv.Atoi(strings.TrimSpace(strings.Split(part, ".")[0]))
		if err != nil || value < 1 || value > length {
			return nil, false
		}
		index := value - 1
		if _, exists := seen[index]; !exists {
			seen[index] = struct{}{}
			result = append(result, index)
		}
	}
	return result, len(result) > 0
}
