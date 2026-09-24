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

func examScoreQuestion(s *utils.SmallSurveyData) (string, *model.Keyboard) {
	name := s.SelectedExamNames[s.ExamScoreStep]
	text := fmt.Sprintf("Ожидаемый балл по предмету «%s»? Введите число от 0 до 100 или нажмите «Пропустить».", name)
	keyboard := model.NewKeyboard()
	keyboard.AddRow().AddMessage("Пропустить")
	return text, keyboard
}

func collectExamScore(s *utils.SmallSurveyData, text string) bool {
	if strings.EqualFold(strings.TrimSpace(text), "Пропустить") {
		s.ExamScoreStep++
		return true
	}
	if s.ExamScoreStep >= len(s.SelectedExamIDs) {
		return false
	}
	score, err := strconv.ParseInt(strings.TrimSpace(text), 10, 16)
	if err != nil || score < 0 || score > 100 {
		return false
	}
	value := int16(score)
	s.SelectedExamScores[s.SelectedExamIDs[s.ExamScoreStep]] = &value
	s.ExamScoreStep++
	return true
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
