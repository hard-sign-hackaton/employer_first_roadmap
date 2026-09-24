package handlers

import (
	"context"
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/utils"
	"strconv"
	"strings"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func gradeKeyboard() *model.Keyboard {
	keyboard := model.NewKeyboard()
	keyboard.AddRow().AddMessage("9").AddMessage("10").AddMessage("11")
	return keyboard
}

func showExamSubjectSelection(ctx maxbot.Context, nextState models.UserState) error {
	userID := ctx.Update().UserID
	s := utils.GetSmallSurvey(userID)
	subjects, err := app.Reference.ListExamSubjects(context.Background())
	if err != nil {
		return ctx.Send("Не удалось получить список предметов ЕГЭ.")
	}
	s.AvailableExamIDs = nil
	s.AvailableExamNames = nil
	s.SelectedExamIDs = nil
	s.SelectedExamNames = nil
	s.SelectedExamScores = nil
	for _, subject := range subjects {
		s.AvailableExamIDs = append(s.AvailableExamIDs, subject.ID)
		s.AvailableExamNames = append(s.AvailableExamNames, subject.Name)
	}
	utils.UpdateUserStateStorage(userID, nextState)
	text, keyboard := examSubjectsQuestion(s)
	return ctx.Send(text, maxbot.WithKeyboard(keyboard))
}

func showExamScorePrompt(ctx maxbot.Context) error {
	keyboard := model.NewKeyboard()
	keyboard.AddRow().AddMessage("Пропустить")
	return ctx.Send("Если знаете ожидаемые баллы, укажите их в формате «1:80,2:75» — номера относятся к выбранным предметам. Или нажмите «Пропустить».", maxbot.WithKeyboard(keyboard))
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
