package handlers

import (
	"context"
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/utils"
	"fmt"
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
	if len(subjects) == 0 {
		return ctx.Send("Список предметов ЕГЭ пока не настроен. Обратитесь к администратору бота.")
	}
	s.AvailableExamIDs = nil
	s.AvailableExamNames = nil
	s.SelectedExamIDs = nil
	s.SelectedExamNames = nil
	s.SelectedExamScores = nil
	for _, subject := range subjects {
		if subject.Name == "Русский язык" {
			s.SelectedExamIDs = append(s.SelectedExamIDs, subject.ID)
			s.SelectedExamNames = append(s.SelectedExamNames, subject.Name)
			continue
		}
		s.AvailableExamIDs = append(s.AvailableExamIDs, subject.ID)
		s.AvailableExamNames = append(s.AvailableExamNames, subject.Name)
	}
	utils.UpdateUserStateStorage(userID, nextState)
	text, keyboard := examSubjectsQuestion(s)
	return ctx.Send(text, maxbot.WithKeyboard(keyboard))
}

func examScoreQuestion(s *utils.SmallSurveyData) (string, *model.Keyboard) {
	name := s.SelectedExamNames[s.ExamScoreStep]
	text := fmt.Sprintf("Какой ожидаемый результат по предмету «%s»? Выберите диапазон. Для подбора используем его верхнюю границу.", name)
	keyboard := model.NewKeyboard()
	keyboard.AddRow().AddMessage("80–100")
	keyboard.AddRow().AddMessage("60–79")
	keyboard.AddRow().AddMessage("0–59")
	return text, keyboard
}

func collectExamScore(s *utils.SmallSurveyData, text string) bool {
	if s.ExamScoreStep >= len(s.SelectedExamIDs) {
		return false
	}
	score, ok := expectedScoreRangeUpperBound(text)
	if !ok {
		return false
	}
	value := score
	s.SelectedExamScores[s.SelectedExamIDs[s.ExamScoreStep]] = &value
	s.ExamScoreStep++
	return true
}

func expectedScoreRangeUpperBound(text string) (int16, bool) {
	switch strings.TrimSpace(text) {
	case "80–100", "80-100":
		return 100, true
	case "60–79", "60-79":
		return 79, true
	case "0–59", "0-59", "Ниже 60":
		return 59, true
	default:
		return 0, false
	}
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
	if len(s.ActualExamNames) > 0 {
		examNames = s.ActualExamNames
	} else if len(s.SelectedExamNames) > 0 {
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

func assessCurrentTrajectoryPath(ctx maxbot.Context) (dto.TrajectoryPathAssessmentResponse, error) {
	s := utils.GetSmallSurvey(ctx.Update().UserID)
	return assessTrajectoryPath(ctx, s.CareerDirectionID)
}

func assessTrajectoryPath(ctx maxbot.Context, careerDirectionID int64) (dto.TrajectoryPathAssessmentResponse, error) {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	request := dto.AssessTrajectoryPathRequest{CareerDirectionID: careerDirectionID}
	if len(s.ActualExamIDs) > 0 {
		request.ExamSubjectIDs = append([]int64(nil), s.ActualExamIDs...)
	} else if s.Grade == 11 && len(s.SelectedExamIDs) > 0 {
		request.ExamSubjectIDs = append([]int64(nil), s.SelectedExamIDs...)
	}
	return app.Trajectory.AssessTrajectoryPath(context.Background(), id, request)
}

func showPathResolution(ctx maxbot.Context, state models.UserState, assessment dto.TrajectoryPathAssessmentResponse) error {
	utils.UpdateUserStateStorage(ctx.Update().UserID, state)
	kb := model.NewKeyboard()
	if assessment.Status == dto.TrajectoryPathStatusRelocationRequired {
		kb.AddRow().AddMessage("Рассмотреть другие регионы")
		kb.AddRow().AddMessage("Изменить направление")
		switch assessment.Issue {
		case dto.TrajectoryPathIssueNoEducationInRegion:
			return ctx.Send("Для выбранного направления в вашем регионе нет подходящих вузов и образовательных программ. Рассмотрите обучение в другом регионе или выберите другое направление.", maxbot.WithKeyboard(kb))
		case dto.TrajectoryPathIssueNoOpportunityInRegion:
			return ctx.Send("Подходящие вузы и программы в вашем регионе есть, но у работодателя пока нет активной практики, стажировки или проекта в регионе вуза. Рассмотрите обучение в другом регионе или выберите другое направление.", maxbot.WithKeyboard(kb))
		default:
			return ctx.Send("Для выбранного направления есть путь до работодателя только при обучении в другом регионе. Рассмотрите обучение в другом регионе или выберите другое направление.", maxbot.WithKeyboard(kb))
		}
	}
	kb.AddRow().AddMessage("Изменить направление")
	kb.AddRow().AddMessage("Сменить работодателя")
	return ctx.Send("Для выбранного направления в каталоге нет полного пути: ЕГЭ → образовательная программа → возможность работодателя. Выберите другое направление или работодателя.", maxbot.WithKeyboard(kb))
}

func showPathRelocationQuestion(ctx maxbot.Context, state models.UserState) error {
	utils.UpdateUserStateStorage(ctx.Update().UserID, state)
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Да").AddMessage("Нет")
	return ctx.Send("Готовы теперь рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(kb))
}

func saveChangedRelocation(ctx maxbot.Context, willingToRelocate bool) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	s.WillingToRelocate = willingToRelocate
	_, err := app.Profile.SaveProfile(context.Background(), id, dto.UpsertProfileRequest{
		Grade:             s.Grade,
		RegionID:          s.RegionID,
		WillingToRelocate: willingToRelocate,
	})
	return err
}
