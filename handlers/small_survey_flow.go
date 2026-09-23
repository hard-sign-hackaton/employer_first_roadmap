package handlers

import (
	"context"
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func handleSmallSurveyMessage(ctx maxbot.Context) error {
	id, text := ctx.Update().UserID, strings.TrimSpace(ctx.Update().Message.Body.Text)
	state, s := utils.GetUserState(id), utils.GetSmallSurvey(id)
	switch state {
	case models.UserStateSmallSurveyWaitingCompanyName:
		n, ok := choice(text, len(s.CompanyIDs))
		if !ok {
			return ctx.Send("Выберите номер компании с клавиатуры.")
		}
		s.CompanyID = s.CompanyIDs[n]
		company, err := app.Trajectory.SelectCompany(context.Background(), id, dto.SelectCompanyRequest{CompanyID: s.CompanyID})
		if err != nil {
			return ctx.Send("Компания не найдена.")
		}
		s.CompanyName = company.Name
		utils.UpdateUserStateStorage(id, models.UserStateSmallSurveyWaitingGrade)
		return ctx.Send("Выбрана компания: " + company.Name + "\n2. В каком вы классе? (9, 10 или 11)")
	case models.UserStateSmallSurveyWaitingGrade:
		grade, err := strconv.ParseInt(text, 10, 16)
		if err != nil || grade < 9 || grade > 11 {
			return ctx.Send("Введите 9, 10 или 11.")
		}
		s.Grade = int16(grade)
		regions, err := app.Reference.ListRegions(context.Background())
		if err != nil {
			return ctx.Send("Не удалось получить регионы.")
		}
		kb := model.NewKeyboard()
		s.RegionIDs = nil
		for i, r := range regions {
			s.RegionIDs = append(s.RegionIDs, r.ID)
			kb.AddRow().AddMessage(fmt.Sprintf("%d. %s", i+1, r.Name))
		}
		utils.UpdateUserStateStorage(id, models.UserStateSmallSurveyWaitingRegion)
		return ctx.Send("3. Выберите регион номером:", maxbot.WithKeyboard(kb))
	case models.UserStateSmallSurveyWaitingRegion:
		n, ok := choice(text, len(s.RegionIDs))
		if !ok {
			return ctx.Send("Выберите номер региона с клавиатуры.")
		}
		s.RegionID = s.RegionIDs[n]
		utils.UpdateUserStateStorage(id, models.UserStateSmallSurveyWaitingRelocation)
		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Да").AddMessage("Нет")
		return ctx.Send("4. Готовы рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(kb))
	case models.UserStateSmallSurveyWaitingRelocation:
		if text != "Да" && text != "Нет" {
			return ctx.Send("Выберите «Да» или «Нет».")
		}
		s.WillingToRelocate = text == "Да"
		if _, err := app.Profile.SaveProfile(context.Background(), id, dto.UpsertProfileRequest{Grade: s.Grade, RegionID: s.RegionID, WillingToRelocate: s.WillingToRelocate}); err != nil {
			return ctx.Send("Не удалось сохранить профиль.")
		}
		if s.Grade == 11 {
			utils.UpdateUserStateStorage(id, models.UserStateSmallSurveyWaitingExamSelection)
			keyboard := model.NewKeyboard()
			keyboard.AddRow().AddMessage("Да").AddMessage("Нет")
			return ctx.Send("5. Вы уже выбрали предметы ЕГЭ?", maxbot.WithKeyboard(keyboard))
		}
		return showDirections(ctx)
	case models.UserStateSmallSurveyWaitingExamSelection:
		if text == "Да" {
			return showExamSubjectSelection(ctx, models.UserStateSmallSurveyWaitingExamSubject)
		}
		if text == "Нет" {
			return showDirections(ctx)
		}
		return ctx.Send("Выберите «Да» или «Нет».")
	case models.UserStateSmallSurveyWaitingExamSubject:
		if !chooseExamSubjects(s, text) {
			return ctx.Send("Введите номера выбранных ЕГЭ через запятую, например: 1,3,5.")
		}
		utils.UpdateUserStateStorage(id, models.UserStateSmallSurveyWaitingExamScore)
		keyboard := model.NewKeyboard()
		keyboard.AddRow().AddMessage("Пропустить")
		return ctx.Send("Если знаете ожидаемые баллы, укажите их в формате «1:80,2:75» — номера относятся к выбранным предметам. Или нажмите «Пропустить».", maxbot.WithKeyboard(keyboard))
	case models.UserStateSmallSurveyWaitingExamScore:
		if !saveExpectedScores(s, text) {
			return ctx.Send("Неверный формат. Используйте «1:80,2:75» или «Пропустить».")
		}
		if err := saveSelectedExamSubjects(ctx); err != nil {
			return ctx.Send("Не удалось сохранить выбранные ЕГЭ.")
		}
		return showDirections(ctx)
	case models.UserStateSmallSurveyWaitingDirection:
		n, ok := choice(text, len(s.DirectionIDs))
		if !ok {
			return ctx.Send("Выберите номер направления.")
		}
		s.CareerDirectionID = s.DirectionIDs[n]
		s.CareerDirectionName = s.DirectionNames[n]
		if s.Grade == 11 && len(s.SelectedExamIDs) > 0 {
			return showGoalConfirmation(ctx, models.UserStateSmallSurveyWaitingGoalConfirmation)
		}
		sets, err := app.Trajectory.GetRecommendedExamSets(context.Background(), dto.GetRecommendedExamSetsRequest{CareerDirectionID: s.CareerDirectionID})
		if err != nil {
			return ctx.Send("Не удалось подобрать наборы ЕГЭ.")
		}
		if len(sets) == 0 {
			return ctx.Send("Для этого направления пока нет наборов ЕГЭ на целевой год. Выберите другое направление.")
		}
		kb := model.NewKeyboard()
		s.ExamSets = nil
		s.ExamSetNames = nil
		for i, set := range sets {
			s.ExamSets = append(s.ExamSets, set.ExamSubjectIDs)
			names := []string{}
			for _, v := range set.Subjects {
				names = append(names, v.Name)
			}
			s.ExamSetNames = append(s.ExamSetNames, names)
			kb.AddRow().AddMessage(fmt.Sprintf("%d. %s", i+1, strings.Join(names, ", ")))
		}
		utils.UpdateUserStateStorage(id, models.UserStateSmallSurveyWaitingExamSet)
		return ctx.Send(fmt.Sprintf("Выберите рекомендуемый набор ЕГЭ. Используем последние доступные правила приёма — %d год:", sets[0].SourceYear), maxbot.WithKeyboard(kb))
	case models.UserStateSmallSurveyWaitingExamSet:
		n, ok := choice(text, len(s.ExamSets))
		if !ok {
			return ctx.Send("Выберите номер набора ЕГЭ.")
		}
		inputs := []dto.UserSubjectInput{}
		for _, subjectID := range s.ExamSets[n] {
			inputs = append(inputs, dto.UserSubjectInput{ExamSubjectID: subjectID, Status: models.SubjectStatusPlanned})
		}
		if _, err := app.Profile.SaveUserSubjects(context.Background(), id, dto.SaveUserSubjectsRequest{Subjects: inputs}); err != nil {
			return ctx.Send("Не удалось сохранить набор ЕГЭ.")
		}
		s.PlannedExamIDs = s.ExamSets[n]
		s.PlannedExamNames = s.ExamSetNames[n]
		return showGoalConfirmation(ctx, models.UserStateSmallSurveyWaitingGoalConfirmation)
	case models.UserStateSmallSurveyWaitingGoalConfirmation:
		if text == "Изменить направление" {
			return showDirections(ctx)
		}
		if text != "Подтвердить" {
			return ctx.Send("Выберите «Подтвердить» или «Изменить направление».")
		}
		goal, err := app.Trajectory.ConfirmGoal(context.Background(), id, dto.ConfirmGoalRequest{CareerDirectionID: s.CareerDirectionID, TargetAdmissionYear: admissionYear(s.Grade)})
		if err != nil {
			return ctx.Send("Не удалось подтвердить цель.")
		}
		roadmap, err := app.Roadmap.CreateRoadmap(context.Background(), id, dto.CreateRoadmapRequest{GoalID: goal.ID})
		if err != nil {
			return ctx.Send("Не удалось сформировать roadmap.")
		}
		utils.UpdateUserStateStorage(id, models.UserStateSurveyCompleted)
		steps := make([]string, 0, len(roadmap.Steps))
		for _, step := range roadmap.Steps {
			steps = append(steps, fmt.Sprintf("%d. %s", step.OrderNo, step.Title))
		}
		if roadmap.NextAction == nil {
			return ctx.Send("Roadmap сформирован:\n" + strings.Join(steps, "\n"))
		}
		return ctx.Send("Цель подтверждена. Roadmap сформирован:\n" + strings.Join(steps, "\n") + "\n\nСледующее действие: " + roadmap.NextAction.Title)
	}
	return nil
}
func showDirections(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	dirs, err := app.Trajectory.GetCareerDirections(context.Background(), id, dto.GetCareerDirectionsRequest{CompanyID: s.CompanyID})
	if err != nil {
		return ctx.Send("Не удалось получить направления.")
	}
	if len(dirs) == 0 {
		if s.Grade == 11 && len(s.SelectedExamIDs) > 0 {
			return ctx.Send("В выбранной компании нет направлений, совместимых с выбранными ЕГЭ и ожидаемыми баллами. Начните опрос заново командой /start, чтобы изменить компанию или ЕГЭ.")
		}
		return ctx.Send("Для выбранной компании пока нет направлений. Начните опрос заново командой /start.")
	}
	kb := model.NewKeyboard()
	s.DirectionIDs = nil
	s.DirectionNames = nil
	for i, d := range dirs {
		s.DirectionIDs = append(s.DirectionIDs, d.ID)
		s.DirectionNames = append(s.DirectionNames, d.Name)
		kb.AddRow().AddMessage(fmt.Sprintf("%d. %s", i+1, d.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateSmallSurveyWaitingDirection)
	return ctx.Send("Выберите карьерное направление:", maxbot.WithKeyboard(kb))
}
func choice(text string, length int) (int, bool) {
	n, err := strconv.Atoi(strings.Split(text, ".")[0])
	return n - 1, err == nil && n > 0 && n <= length
}
func admissionYear(grade int16) int16 {
	year := time.Now().Year()
	if time.Now().Month() >= time.September {
		year++
	}
	return int16(year + int(11-grade))
}
