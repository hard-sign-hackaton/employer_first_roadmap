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

func handleBigSurveyMessage(ctx maxbot.Context) error {
	id, text := ctx.Update().UserID, strings.TrimSpace(ctx.Update().Message.Body.Text)
	state, s := utils.GetUserState(id), utils.GetSmallSurvey(id)
	switch state {
	case models.UserStateBigSurveyWaitingGrade:
		grade, err := strconv.ParseInt(text, 10, 16)
		if err != nil || grade < 9 || grade > 11 {
			return ctx.Send("Введите 9, 10 или 11.")
		}
		s.Grade = int16(grade)
		return showBigRegions(ctx)
	case models.UserStateBigSurveyWaitingRegion:
		index, ok := choice(text, len(s.RegionIDs))
		if !ok {
			return ctx.Send("Выберите номер региона с клавиатуры.")
		}
		s.RegionID = s.RegionIDs[index]
		utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingRelocation)
		keyboard := model.NewKeyboard()
		keyboard.AddRow().AddMessage("Да").AddMessage("Нет")
		return ctx.Send("3. Готовы рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(keyboard))
	case models.UserStateBigSurveyWaitingRelocation:
		if text != "Да" && text != "Нет" {
			return ctx.Send("Выберите «Да» или «Нет».")
		}
		s.WillingToRelocate = text == "Да"
		if _, err := app.Profile.SaveProfile(context.Background(), id, dto.UpsertProfileRequest{Grade: s.Grade, RegionID: s.RegionID, WillingToRelocate: s.WillingToRelocate}); err != nil {
			return ctx.Send("Не удалось сохранить профиль.")
		}
		if s.Grade == 11 {
			utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingExamSelection)
			keyboard := model.NewKeyboard()
			keyboard.AddRow().AddMessage("Да").AddMessage("Нет")
			return ctx.Send("4. Вы уже выбрали предметы ЕГЭ?", maxbot.WithKeyboard(keyboard))
		}
		return showFullSurveyActivities(ctx)
	case models.UserStateBigSurveyWaitingExamSelection:
		if text == "Да" {
			return showExamSubjectSelection(ctx, models.UserStateBigSurveyWaitingExamSubject)
		}
		if text == "Нет" {
			return showFullSurveyActivities(ctx)
		}
		return ctx.Send("Выберите «Да» или «Нет».")
	case models.UserStateBigSurveyWaitingExamSubject:
		if !chooseExamSubjects(s, text) {
			return ctx.Send("Введите номера выбранных ЕГЭ через запятую, например: 1,3,5.")
		}
		utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingExamScore)
		keyboard := model.NewKeyboard()
		keyboard.AddRow().AddMessage("Пропустить")
		return ctx.Send("Если знаете ожидаемые баллы, укажите их в формате «1:80,2:75». Или нажмите «Пропустить».", maxbot.WithKeyboard(keyboard))
	case models.UserStateBigSurveyWaitingExamScore:
		if !saveExpectedScores(s, text) {
			return ctx.Send("Неверный формат. Используйте «1:80,2:75» или «Пропустить».")
		}
		if err := saveSelectedExamSubjects(ctx); err != nil {
			return ctx.Send("Не удалось сохранить выбранные ЕГЭ.")
		}
		return showFullSurveyActivities(ctx)
	case models.UserStateBigSurveyWaitingInterest:
		index, ok := choice(text, len(s.ActivityTagIDs))
		if !ok {
			return ctx.Send("Выберите номер интереса с клавиатуры.")
		}
		s.SelectedActivityTagIDs = []int64{s.ActivityTagIDs[index]}
		return showFullSurveySchoolSubjects(ctx)
	case models.UserStateBigSurveyWaitingSubjects:
		indexes, ok := multipleChoices(text, len(s.AvailableExamIDs))
		if !ok {
			return ctx.Send("Введите номера школьных предметов через запятую, например: 2,3,6.")
		}
		s.SelectedSchoolSubjectNames = nil
		for _, index := range indexes {
			s.SelectedSchoolSubjectNames = append(s.SelectedSchoolSubjectNames, s.AvailableExamNames[index])
		}
		if err := saveFullSurveyInterests(ctx); err != nil {
			return ctx.Send("Не удалось сохранить интересы опроса.")
		}
		return showRecommendedCompanies(ctx)
	case models.UserStateBigSurveyWaitingCompany:
		index, ok := choice(text, len(s.CompanyIDs))
		if !ok {
			return ctx.Send("Выберите номер компании с клавиатуры.")
		}
		s.CompanyID = s.CompanyIDs[index]
		company, err := app.Trajectory.SelectCompany(context.Background(), id, dto.SelectCompanyRequest{CompanyID: s.CompanyID})
		if err != nil {
			return ctx.Send("Компания не найдена.")
		}
		s.CompanyName = company.Name
		return showBigDirections(ctx)
	case models.UserStateBigSurveyWaitingDirection:
		index, ok := choice(text, len(s.DirectionIDs))
		if !ok {
			return ctx.Send("Выберите номер направления.")
		}
		s.CareerDirectionID = s.DirectionIDs[index]
		s.CareerDirectionName = s.DirectionNames[index]
		if s.Grade == 11 && len(s.SelectedExamIDs) > 0 {
			return showGoalConfirmation(ctx, models.UserStateBigSurveyWaitingGoalConfirmation)
		}
		return showBigRecommendedExamSets(ctx)
	case models.UserStateBigSurveyWaitingExamSet:
		index, ok := choice(text, len(s.ExamSets))
		if !ok {
			return ctx.Send("Выберите номер набора ЕГЭ.")
		}
		inputs := make([]dto.UserSubjectInput, 0, len(s.ExamSets[index]))
		for _, subjectID := range s.ExamSets[index] {
			inputs = append(inputs, dto.UserSubjectInput{ExamSubjectID: subjectID, Status: models.SubjectStatusPlanned})
		}
		if _, err := app.Profile.SaveUserSubjects(context.Background(), id, dto.SaveUserSubjectsRequest{Subjects: inputs}); err != nil {
			return ctx.Send("Не удалось сохранить набор ЕГЭ.")
		}
		s.PlannedExamIDs = s.ExamSets[index]
		s.PlannedExamNames = s.ExamSetNames[index]
		return showGoalConfirmation(ctx, models.UserStateBigSurveyWaitingGoalConfirmation)
	case models.UserStateBigSurveyWaitingGoalConfirmation:
		if text == "Изменить направление" {
			return showBigDirections(ctx)
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
		return sendRoadmap(ctx, roadmap)
	}
	return nil
}

func showBigRegions(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	regions, err := app.Reference.ListRegions(context.Background())
	if err != nil {
		return ctx.Send("Не удалось получить регионы.")
	}
	s.RegionIDs = nil
	keyboard := model.NewKeyboard()
	for index, region := range regions {
		s.RegionIDs = append(s.RegionIDs, region.ID)
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, region.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingRegion)
	return ctx.Send("2. Выберите регион номером:", maxbot.WithKeyboard(keyboard))
}

func showFullSurveyActivities(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	tags, err := app.Reference.ListInterestTags(context.Background())
	if err != nil {
		return ctx.Send("Не удалось получить список интересов.")
	}
	s.ActivityTagIDs = nil
	s.ActivityTagNames = nil
	keyboard := model.NewKeyboard()
	actionTags := map[string]bool{
		"Программирование": true,
		"Аналитика":        true,
		"Инженерия":        true,
		"Коммуникация":     true,
		"Исследования":     true,
	}
	for _, tag := range tags {
		if !actionTags[tag.Name] {
			continue
		}
		index := len(s.ActivityTagIDs)
		s.ActivityTagIDs = append(s.ActivityTagIDs, tag.InterestTagID)
		s.ActivityTagNames = append(s.ActivityTagNames, tag.Name)
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, tag.Name))
	}
	s.ActivityTagIDs = append(s.ActivityTagIDs, 0)
	s.ActivityTagNames = append(s.ActivityTagNames, "Другое")
	keyboard.AddRow().AddMessage(fmt.Sprintf("%d. Другое", len(s.ActivityTagIDs)))
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingInterest)
	return ctx.Send("Что вам интереснее всего делать? Выберите один вариант:", maxbot.WithKeyboard(keyboard))
}

func showFullSurveySchoolSubjects(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	subjects, err := app.Reference.ListExamSubjects(context.Background())
	if err != nil {
		return ctx.Send("Не удалось получить школьные предметы.")
	}
	s.AvailableExamIDs = nil
	s.AvailableExamNames = nil
	keyboard := model.NewKeyboard()
	for index, subject := range subjects {
		s.AvailableExamIDs = append(s.AvailableExamIDs, subject.ID)
		s.AvailableExamNames = append(s.AvailableExamNames, subject.Name)
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, subject.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingSubjects)
	return ctx.Send("Какие школьные предметы вам интересны? Выберите номера через запятую:", maxbot.WithKeyboard(keyboard))
}

func saveFullSurveyInterests(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	weights := make(map[int64]float64)
	for _, tagID := range s.SelectedActivityTagIDs {
		if tagID > 0 {
			weights[tagID] += 2
		}
	}
	tags, err := app.Reference.ListInterestTags(context.Background())
	if err != nil {
		return err
	}
	byName := make(map[string]int64, len(tags))
	for _, tag := range tags {
		byName[tag.Name] = tag.InterestTagID
	}
	for _, subject := range s.SelectedSchoolSubjectNames {
		if tagName := schoolSubjectTag(subject); tagName != "" {
			weights[byName[tagName]]++
		}
	}
	inputs := make([]dto.InterestInput, 0, len(weights))
	for tagID, weight := range weights {
		if tagID > 0 {
			inputs = append(inputs, dto.InterestInput{InterestTagID: tagID, Weight: weight})
		}
	}
	_, err = app.Profile.SaveSurveyInterests(context.Background(), id, dto.SaveSurveyInterestsRequest{Interests: inputs})
	return err
}

func schoolSubjectTag(subject string) string {
	switch subject {
	case "Математика (профильная)":
		return "Математика"
	case "Информатика":
		return "Программирование"
	case "Физика":
		return "Физика"
	case "Обществознание":
		return "Коммуникация"
	case "Английский язык":
		return "Английский язык"
	case "Химия", "Биология":
		return "Исследования"
	default:
		return ""
	}
}

func showRecommendedCompanies(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	companies, err := app.Trajectory.RecommendCompanies(context.Background(), id, dto.RecommendCompaniesRequest{Limit: 10})
	if err != nil {
		return ctx.Send("Не удалось подобрать компании.")
	}
	if len(companies) == 0 {
		if s.Grade == 11 && len(s.SelectedExamIDs) > 0 {
			utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingExamSelection)
			keyboard := model.NewKeyboard()
			keyboard.AddRow().AddMessage("Да").AddMessage("Нет")
			return ctx.Send("С выбранными ЕГЭ подходящих направлений не найдено. Хотите изменить набор ЕГЭ?", maxbot.WithKeyboard(keyboard))
		}
		return ctx.Send("Подходящих компаний не найдено. Начните опрос заново командой /start.")
	}
	s.CompanyIDs = nil
	keyboard := model.NewKeyboard()
	lines := make([]string, 0, len(companies))
	for index, company := range companies {
		s.CompanyIDs = append(s.CompanyIDs, company.ID)
		explanation := ""
		if len(company.Reasons) > 0 {
			explanation = " — " + company.Reasons[0]
		}
		lines = append(lines, fmt.Sprintf("%d. %s%s", index+1, company.Name, explanation))
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, company.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingCompany)
	return ctx.Send("Подходящие компании подобраны по вашим интересам:\n"+strings.Join(lines, "\n")+"\n\nВыберите компанию номером:", maxbot.WithKeyboard(keyboard))
}

func showBigDirections(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	directions, err := app.Trajectory.GetCareerDirections(context.Background(), id, dto.GetCareerDirectionsRequest{CompanyID: s.CompanyID})
	if err != nil {
		return ctx.Send("Не удалось получить направления компании.")
	}
	if len(directions) == 0 {
		return ctx.Send("Для выбранной компании нет направлений, совместимых с выбранными ЕГЭ. Вернитесь к выбору ЕГЭ командой /start.")
	}
	s.DirectionIDs = nil
	s.DirectionNames = nil
	keyboard := model.NewKeyboard()
	for index, direction := range directions {
		s.DirectionIDs = append(s.DirectionIDs, direction.ID)
		s.DirectionNames = append(s.DirectionNames, direction.Name)
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, direction.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingDirection)
	return ctx.Send("Выберите карьерное направление:", maxbot.WithKeyboard(keyboard))
}

func showBigRecommendedExamSets(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	sets, err := app.Trajectory.GetRecommendedExamSets(context.Background(), dto.GetRecommendedExamSetsRequest{CareerDirectionID: s.CareerDirectionID})
	if err != nil {
		return ctx.Send("Не удалось подобрать наборы ЕГЭ.")
	}
	if len(sets) == 0 {
		return ctx.Send("Для направления пока нет опубликованных наборов ЕГЭ. Выберите другое направление.")
	}
	s.ExamSets = nil
	s.ExamSetNames = nil
	keyboard := model.NewKeyboard()
	for index, set := range sets {
		s.ExamSets = append(s.ExamSets, set.ExamSubjectIDs)
		names := make([]string, 0, len(set.Subjects))
		for _, subject := range set.Subjects {
			names = append(names, subject.Name)
		}
		s.ExamSetNames = append(s.ExamSetNames, names)
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, strings.Join(names, ", ")))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingExamSet)
	return ctx.Send(fmt.Sprintf("Выберите рекомендуемый набор ЕГЭ. Используем последние доступные правила приёма — %d год:", sets[0].SourceYear), maxbot.WithKeyboard(keyboard))
}

func sendRoadmap(ctx maxbot.Context, roadmap dto.RoadmapResponse) error {
	steps := make([]string, 0, len(roadmap.Steps))
	for _, step := range roadmap.Steps {
		steps = append(steps, fmt.Sprintf("%d. %s", step.OrderNo, step.Title))
	}
	if roadmap.NextAction == nil {
		return ctx.Send("Цель подтверждена. Roadmap сформирован:\n" + strings.Join(steps, "\n"))
	}
	return ctx.Send("Цель подтверждена. Roadmap сформирован:\n" + strings.Join(steps, "\n") + "\n\nСледующее действие: " + roadmap.NextAction.Title)
}
