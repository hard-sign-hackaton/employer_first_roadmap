package handlers

import (
	"context"
	"efr_bot/dto"
	"efr_bot/models"
	"efr_bot/utils"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

const (
	subjectToggleCallback     = "/subject_toggle"
	subjectsDoneCallback      = "/subjects_done"
	examSubjectToggleCallback = "/exam_subject_toggle"
	examSubjectsDoneCallback  = "/exam_subjects_done"
)

func handleBigSurveyMessage(ctx maxbot.Context) error {
	id, text := ctx.Update().UserID, strings.TrimSpace(ctx.Update().Message.Body.Text)
	state, s := utils.GetUserState(id), utils.GetSmallSurvey(id)
	switch state {
	case models.UserStateBigSurveyWaitingGrade:
		grade, err := strconv.ParseInt(text, 10, 16)
		if err != nil || grade < 9 || grade > 11 {
			return resendBigGrade(ctx, "Введите 9, 10 или 11.")
		}
		s.Grade = int16(grade)
		return showBigRegions(ctx)
	case models.UserStateBigSurveyWaitingRegion:
		index, ok := choice(text, len(s.RegionIDs))
		if !ok {
			return resendBigRegions(ctx, "Выберите номер региона с клавиатуры.")
		}
		s.RegionID = s.RegionIDs[index]
		utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingRelocation)
		keyboard := model.NewKeyboard()
		keyboard.AddRow().AddMessage("Да").AddMessage("Нет")
		return ctx.Send("3. Готовы рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(keyboard))
	case models.UserStateBigSurveyWaitingRelocation:
		if text != "Да" && text != "Нет" {
			return resendBigRelocation(ctx, "Выберите «Да» или «Нет».")
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
		return resendBigExamSelection(ctx, "Выберите «Да» или «Нет».")
	case models.UserStateBigSurveyWaitingExamSubject:
		return resendExamSubjects(ctx, "Выберите предметы ЕГЭ кнопками ниже и нажмите «Готово».")
	case models.UserStateBigSurveyWaitingExamScore:
		if !collectExamScore(s, text) {
			return resendExamScore(ctx, "Выберите один из диапазонов баллов кнопками ниже.")
		}
		if s.ExamScoreStep < len(s.SelectedExamIDs) {
			text, keyboard := examScoreQuestion(s)
			return ctx.Send(text, maxbot.WithKeyboard(keyboard))
		}
		if err := saveSelectedExamSubjects(ctx); err != nil {
			return ctx.Send("Не удалось сохранить выбранные ЕГЭ.")
		}
		return showFullSurveyActivities(ctx)
	case models.UserStateBigSurveyWaitingInterest:
		index, ok := choice(text, len(s.ActivityTagIDs))
		if !ok {
			return resendBigActivities(ctx, "Выберите номер интереса с клавиатуры.")
		}
		s.SelectedActivityKey = s.ActivityTagNames[index]
		return showFullSurveySchoolSubjects(ctx)
	case models.UserStateBigSurveyWaitingSubjects:
		return resendSchoolSubjects(ctx, "Выберите школьные предметы кнопками ниже и нажмите «Готово».")
	case models.UserStateBigSurveyWaitingCompany:
		index, ok := choice(text, len(s.CompanyIDs))
		if !ok {
			return resendRecommendedCompanies(ctx, "Выберите номер компании с клавиатуры.")
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
			return resendBigDirections(ctx, "Выберите номер направления.")
		}
		s.CareerDirectionID = s.DirectionIDs[index]
		s.CareerDirectionName = s.DirectionNames[index]
		return continueBigTrajectoryAfterDirection(ctx)
	case models.UserStateBigSurveyWaitingPathResolution:
		switch text {
		case "Рассмотреть другие регионы":
			return showPathRelocationQuestion(ctx, models.UserStateBigSurveyWaitingPathRelocation)
		case "Изменить направление":
			return showBigDirections(ctx)
		case "Сменить работодателя":
			return showRecommendedCompanies(ctx)
		}
		return ctx.Send("Выберите действие с клавиатуры.")
	case models.UserStateBigSurveyWaitingPathRelocation:
		if text != "Да" && text != "Нет" {
			return showPathRelocationQuestion(ctx, models.UserStateBigSurveyWaitingPathRelocation)
		}
		if text == "Нет" {
			return showBigPathResolution(ctx, dto.TrajectoryPathAssessmentResponse{Status: dto.TrajectoryPathStatusRelocationRequired})
		}
		if err := saveChangedRelocation(ctx, true); err != nil {
			return ctx.Send("Не удалось сохранить решение о переезде.")
		}
		return continueBigTrajectoryAfterDirection(ctx)
	case models.UserStateBigSurveyWaitingCompanyPathResolution:
		switch text {
		case "Рассмотреть другие регионы":
			if err := saveChangedRelocation(ctx, true); err != nil {
				return ctx.Send("Не удалось сохранить решение о переезде.")
			}
			return showRecommendedCompanies(ctx)
		case "Изменить интересы":
			return showFullSurveyActivities(ctx)
		case "Начать новый опрос":
			return CallBigSurvey(ctx)
		}
		return ctx.Send("Выберите действие с клавиатуры.")
	case models.UserStateBigSurveyWaitingExamSet:
		index, ok := choice(text, len(s.ExamSets))
		if !ok {
			return resendBigExamSets(ctx, "Выберите номер набора ЕГЭ.")
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
			if err := ctx.Send("Выберите «Подтвердить» или «Изменить направление»."); err != nil {
				return err
			}
			return showGoalConfirmation(ctx, models.UserStateBigSurveyWaitingGoalConfirmation)
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
	if len(regions) == 0 {
		return ctx.Send("Список регионов пока не настроен. Обратитесь к администратору бота.")
	}
	s.RegionIDs = nil
	keyboard := model.NewKeyboard()
	for index, region := range regions {
		s.RegionIDs = append(s.RegionIDs, region.ID)
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, region.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingRegion)
	return ctx.Send("2. Выберите регион:", maxbot.WithKeyboard(keyboard))
}

func resendBigGrade(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return ctx.Send("1. Выберите класс:", maxbot.WithKeyboard(gradeKeyboard()))
}

func resendBigRegions(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showBigRegions(ctx)
}

func resendBigRelocation(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Да").AddMessage("Нет")
	return ctx.Send("3. Готовы рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(kb))
}

func resendBigExamSelection(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Да").AddMessage("Нет")
	return ctx.Send("4. Вы уже выбрали предметы ЕГЭ?", maxbot.WithKeyboard(kb))
}

func resendBigActivities(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showFullSurveyActivities(ctx)
}

func resendSchoolSubjects(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	text, kb := schoolSubjectsQuestion(utils.GetSmallSurvey(ctx.Update().UserID))
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func resendExamSubjects(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	text, kb := examSubjectsQuestion(utils.GetSmallSurvey(ctx.Update().UserID))
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func resendExamScore(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	text, kb := examScoreQuestion(utils.GetSmallSurvey(ctx.Update().UserID))
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func resendRecommendedCompanies(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showRecommendedCompanies(ctx)
}

func resendBigDirections(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showBigDirections(ctx)
}

func resendBigExamSets(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showBigRecommendedExamSets(ctx)
}

func showFullSurveyActivities(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	s.ActivityTagIDs = nil
	s.ActivityTagNames = nil
	keyboard := model.NewKeyboard()
	for index, profile := range surveyActivityProfiles {
		s.ActivityTagIDs = append(s.ActivityTagIDs, int64(index+1))
		s.ActivityTagNames = append(s.ActivityTagNames, profile.Key)
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, profile.Label))
	}
	s.ActivityTagIDs = append(s.ActivityTagIDs, 0)
	s.ActivityTagNames = append(s.ActivityTagNames, "unknown")
	keyboard.AddRow().AddMessage(fmt.Sprintf("%d. Пока не знаю", len(s.ActivityTagIDs)))
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingInterest)
	return ctx.Send("Что вам интереснее всего делать? Это поможет подобрать направления, но не ограничит ваш выбор. Выберите один вариант:", maxbot.WithKeyboard(keyboard))
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
	for _, subject := range subjects {
		if subject.Name == "Математика (профильная)" {
			continue
		}
		s.AvailableExamIDs = append(s.AvailableExamIDs, subject.ID)
		s.AvailableExamNames = append(s.AvailableExamNames, subject.Name)
	}
	s.SelectedSchoolSubjectIDs = nil
	s.SelectedSchoolSubjectNames = nil
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingSubjects)

	text, keyboard := schoolSubjectsQuestion(s)
	return ctx.Send(text, maxbot.WithKeyboard(keyboard))
}

func schoolSubjectsQuestion(s *utils.SmallSurveyData) (string, *model.Keyboard) {
	keyboard := model.NewKeyboard()
	for index, subjectID := range s.AvailableExamIDs {
		label := fmt.Sprintf("%d. %s", index+1, s.AvailableExamNames[index])
		if isSchoolSubjectSelected(s, subjectID) {
			label = "✓ " + label
		}
		keyboard.AddRow().AddCallBack(label, fmt.Sprintf("%s:%d", subjectToggleCallback, subjectID))
	}
	keyboard.AddRow().AddCallBack("Готово", subjectsDoneCallback)

	text := "Какие школьные предметы вам нравятся? Можно выбрать несколько."
	if len(s.SelectedSchoolSubjectNames) > 0 {
		text += "\n\nВыбрано: " + strings.Join(s.SelectedSchoolSubjectNames, ", ")
	}
	return text, keyboard
}

func isSchoolSubjectSelected(s *utils.SmallSurveyData, subjectID int64) bool {
	if slices.Contains(s.SelectedSchoolSubjectIDs, subjectID) {
		return true
	}
	return false
}

func toggleSchoolSubject(s *utils.SmallSurveyData, subjectID int64) {
	for i, id := range s.SelectedSchoolSubjectIDs {
		if id == subjectID {
			s.SelectedSchoolSubjectIDs = append(s.SelectedSchoolSubjectIDs[:i], s.SelectedSchoolSubjectIDs[i+1:]...)
			s.SelectedSchoolSubjectNames = append(s.SelectedSchoolSubjectNames[:i], s.SelectedSchoolSubjectNames[i+1:]...)
			return
		}
	}
	name := ""
	for i, id := range s.AvailableExamIDs {
		if id == subjectID {
			name = s.AvailableExamNames[i]
			break
		}
	}
	s.SelectedSchoolSubjectIDs = append(s.SelectedSchoolSubjectIDs, subjectID)
	s.SelectedSchoolSubjectNames = append(s.SelectedSchoolSubjectNames, name)
}

func SubjectToggle(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	if utils.GetUserState(id) != models.UserStateBigSurveyWaitingSubjects {
		return ctx.Answer("Этот выбор уже завершён. Начните заново командой /start.")
	}
	payload := ctx.Update().GetCallbackPayload()
	subjectID, err := strconv.ParseInt(payload.Param, 10, 64)
	if err != nil {
		return nil
	}
	s := utils.GetSmallSurvey(id)
	subjectName := ""
	found := false
	for i, availableID := range s.AvailableExamIDs {
		if availableID == subjectID {
			subjectName = s.AvailableExamNames[i]
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	toggleSchoolSubject(s, subjectID)
	if isSchoolSubjectSelected(s, subjectID) {
		_ = ctx.Answer("✓ Выбрано: " + subjectName)
	} else {
		_ = ctx.Answer("Снято: " + subjectName)
	}
	text, keyboard := schoolSubjectsQuestion(s)
	return ctx.Edit(text, maxbot.WithKeyboard(keyboard))
}

func SubjectsDone(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	if utils.GetUserState(id) != models.UserStateBigSurveyWaitingSubjects {
		return ctx.Answer("Этот выбор уже завершён. Начните заново командой /start.")
	}
	s := utils.GetSmallSurvey(id)
	if len(s.SelectedSchoolSubjectIDs) == 0 {
		if err := ctx.Send("Выберите хотя бы один предмет."); err != nil {
			return err
		}
		text, keyboard := schoolSubjectsQuestion(s)
		return ctx.Edit(text, maxbot.WithKeyboard(keyboard))
	}
	if err := saveFullSurveyInterests(ctx); err != nil {
		return ctx.Answer("Не удалось сохранить интересы опроса.")
	}
	_ = ctx.Answer(fmt.Sprintf("✓ Выбрано предметов: %d", len(s.SelectedSchoolSubjectIDs)))
	return showRecommendedCompanies(ctx)
}

func examSubjectsQuestion(s *utils.SmallSurveyData) (string, *model.Keyboard) {
	keyboard := model.NewKeyboard()
	for index, subjectID := range s.AvailableExamIDs {
		label := fmt.Sprintf("%d. %s", index+1, s.AvailableExamNames[index])
		if isExamSubjectSelected(s, subjectID) {
			label = "✓ " + label
		}
		keyboard.AddRow().AddCallBack(label, fmt.Sprintf("%s:%d", examSubjectToggleCallback, subjectID))
	}
	keyboard.AddRow().AddCallBack("Готово", examSubjectsDoneCallback)

	text := "Какие предметы ЕГЭ вы уже выбрали? Нажимайте на предметы — выбранные помечаются галочкой. Когда закончите, нажмите «Готово»."
	if len(s.SelectedExamNames) > 0 {
		text += "\n\nВыбрано: " + strings.Join(s.SelectedExamNames, ", ")
	}
	return text, keyboard
}

func isExamSubjectSelected(s *utils.SmallSurveyData, subjectID int64) bool {
	return slices.Contains(s.SelectedExamIDs, subjectID)
}

func toggleExamSubject(s *utils.SmallSurveyData, subjectID int64) {
	for i, id := range s.SelectedExamIDs {
		if id == subjectID {
			s.SelectedExamIDs = append(s.SelectedExamIDs[:i], s.SelectedExamIDs[i+1:]...)
			s.SelectedExamNames = append(s.SelectedExamNames[:i], s.SelectedExamNames[i+1:]...)
			return
		}
	}
	name := ""
	for i, id := range s.AvailableExamIDs {
		if id == subjectID {
			name = s.AvailableExamNames[i]
			break
		}
	}
	s.SelectedExamIDs = append(s.SelectedExamIDs, subjectID)
	s.SelectedExamNames = append(s.SelectedExamNames, name)
}

func ExamSubjectToggle(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	state := utils.GetUserState(id)
	if state != models.UserStateSmallSurveyWaitingExamSubject && state != models.UserStateBigSurveyWaitingExamSubject && state != models.UserStateRoadmapExamChoiceSubjects {
		return ctx.Answer("Этот выбор уже завершён. Начните заново командой /start.")
	}
	payload := ctx.Update().GetCallbackPayload()
	subjectID, err := strconv.ParseInt(payload.Param, 10, 64)
	if err != nil {
		return nil
	}
	s := utils.GetSmallSurvey(id)
	subjectName := ""
	found := false
	for i, availableID := range s.AvailableExamIDs {
		if availableID == subjectID {
			subjectName = s.AvailableExamNames[i]
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	toggleExamSubject(s, subjectID)
	if isExamSubjectSelected(s, subjectID) {
		_ = ctx.Answer("✓ Выбрано: " + subjectName)
	} else {
		_ = ctx.Answer("Снято: " + subjectName)
	}
	text, keyboard := examSubjectsQuestion(s)
	return ctx.Edit(text, maxbot.WithKeyboard(keyboard))
}

func ExamSubjectsDone(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	state := utils.GetUserState(id)
	var nextState models.UserState
	s := utils.GetSmallSurvey(id)
	if len(s.SelectedExamIDs) == 0 {
		if err := ctx.Send("Выберите хотя бы один предмет."); err != nil {
			return err
		}
		text, keyboard := examSubjectsQuestion(s)
		return ctx.Edit(text, maxbot.WithKeyboard(keyboard))
	}
	switch state {
	case models.UserStateSmallSurveyWaitingExamSubject:
		nextState = models.UserStateSmallSurveyWaitingExamScore
	case models.UserStateBigSurveyWaitingExamSubject:
		nextState = models.UserStateBigSurveyWaitingExamScore
	case models.UserStateRoadmapExamChoiceSubjects:
		status := models.SubjectStatusSelected
		if s.Grade < 11 {
			status = models.SubjectStatusPlanned
		}
		inputs := make([]dto.UserSubjectInput, 0, len(s.SelectedExamIDs))
		for _, subjectID := range s.SelectedExamIDs {
			inputs = append(inputs, dto.UserSubjectInput{ExamSubjectID: subjectID, Status: status})
		}
		if _, err := app.Profile.SaveUserSubjects(context.Background(), id, dto.SaveUserSubjectsRequest{Subjects: inputs}); err != nil {
			return ctx.Answer("Не удалось сохранить обновлённый набор ЕГЭ.")
		}
		_ = ctx.Answer(fmt.Sprintf("✓ Набор ЕГЭ обновлён: %d предмет(ов)", len(s.SelectedExamIDs)))
		return showRoadmapExamChoice(ctx)
	default:
		return ctx.Answer("Этот выбор уже завершён. Начните заново командой /start.")
	}
	s.SelectedExamScores = make(map[int64]*int16)
	s.ExamScoreStep = 0
	utils.UpdateUserStateStorage(id, nextState)
	_ = ctx.Answer(fmt.Sprintf("✓ Выбрано предметов: %d", len(s.SelectedExamIDs)))
	text, keyboard := examScoreQuestion(s)
	return ctx.Send(text, maxbot.WithKeyboard(keyboard))
}

func saveFullSurveyInterests(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	tags, err := app.Reference.ListInterestTags(context.Background())
	if err != nil {
		return err
	}
	byName := make(map[string]int64, len(tags))
	for _, tag := range tags {
		byName[tag.Name] = tag.InterestTagID
	}
	weights := make(map[int64]float64)
	if profile, ok := surveyActivityProfileByKey(s.SelectedActivityKey); ok {
		for tagName, weight := range profile.TagWeights {
			if tagID := byName[tagName]; tagID > 0 {
				weights[tagID] += weight
			}
		}
	}
	for _, subject := range s.SelectedSchoolSubjectNames {
		for tagName, weight := range schoolSubjectWeights(subject) {
			if tagID := byName[tagName]; tagID > 0 {
				weights[tagID] += weight
			}
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

type surveyActivityProfile struct {
	Key        string
	Label      string
	TagWeights map[string]float64
}

var surveyActivityProfiles = []surveyActivityProfile{
	{Key: "technology", Label: "Создавать технологии и работать с техникой", TagWeights: map[string]float64{"Программирование": 2, "Инженерия": 1.8, "Производство": 1, "Математика": .8, "Физика": .8}},
	{Key: "research", Label: "Исследовать и находить закономерности", TagWeights: map[string]float64{"Исследования": 2, "Аналитика": 1.4, "Математика": 1, "Физика": .8, "Химия": .8, "Биология": .8, "Экология": .8}},
	{Key: "people", Label: "Помогать людям и общаться", TagWeights: map[string]float64{"Коммуникация": 2, "Медицина": 1.4, "Английский язык": .6}},
	{Key: "projects", Label: "Организовывать и развивать проекты", TagWeights: map[string]float64{"Управление продуктом": 2, "Аналитика": 1.3, "Логистика": 1.3, "Коммуникация": .8}},
	{Key: "creative", Label: "Придумывать, писать и создавать визуальное", TagWeights: map[string]float64{"Дизайн": 2, "Техническая документация": 1.5, "Коммуникация": 1.2, "Английский язык": .8}},
}

func surveyActivityProfileByKey(key string) (surveyActivityProfile, bool) {
	for _, profile := range surveyActivityProfiles {
		if profile.Key == key {
			return profile, true
		}
	}
	return surveyActivityProfile{}, false
}

// schoolSubjectWeights добавляет предметы только как слабый сигнал интереса.
// Эти веса никогда не участвуют в подборе или проверке комбинаций ЕГЭ.
func schoolSubjectWeights(subject string) map[string]float64 {
	switch subject {
	case "Русский язык":
		return map[string]float64{"Коммуникация": .4, "Техническая документация": .2}
	case "Математика":
		return map[string]float64{"Математика": .4, "Аналитика": .2}
	case "Информатика":
		return map[string]float64{"Программирование": .4, "Аналитика": .15}
	case "Физика":
		return map[string]float64{"Физика": .4, "Инженерия": .2}
	case "Обществознание":
		return map[string]float64{"Коммуникация": .4, "Управление продуктом": .15}
	case "Английский язык":
		return map[string]float64{"Английский язык": .4, "Коммуникация": .15}
	case "Химия":
		return map[string]float64{"Химия": .4, "Исследования": .15}
	case "Биология":
		return map[string]float64{"Биология": .4, "Медицина": .2}
	case "География":
		return map[string]float64{"Экология": .4, "Логистика": .15}
	case "История":
		return map[string]float64{"Коммуникация": .35, "Исследования": .15}
	case "Литература":
		return map[string]float64{"Коммуникация": .35, "Техническая документация": .2}
	default:
		return nil
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
		diagnosis, diagnosisErr := app.Trajectory.DiagnoseCompanyRecommendations(context.Background(), id)
		if diagnosisErr != nil {
			return ctx.Send("Не удалось определить, почему не нашлись компании. Попробуйте позже.")
		}
		utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingCompanyPathResolution)
		keyboard := model.NewKeyboard()
		switch diagnosis.Issue {
		case dto.CompanyRecommendationIssueRelocationRequired:
			keyboard.AddRow().AddMessage("Рассмотреть другие регионы")
			keyboard.AddRow().AddMessage("Изменить интересы")
			return ctx.Send("По текущим данным каталога в вашем регионе нет ни одной полной траектории до работодателя: не находится связка направления, ОП/вуза и возможности работодателя. В других регионах такие траектории есть.", maxbot.WithKeyboard(keyboard))
		default:
			keyboard.AddRow().AddMessage("Изменить интересы")
			keyboard.AddRow().AddMessage("Начать новый опрос")
			return ctx.Send("По текущим данным каталога не найдено ни одной полной траектории: направление → ЕГЭ → ОП/вуз → возможность работодателя. Измените интересы или начните новый опрос.", maxbot.WithKeyboard(keyboard))
		}
	}
	s.CompanyIDs = nil
	keyboard := model.NewKeyboard()
	lines := make([]string, 0, len(companies))
	for index, company := range companies {
		s.CompanyIDs = append(s.CompanyIDs, company.ID)
		explanation := ""
		if len(company.Reasons) > 0 {
			explanation = "\nПочему рекомендована: " + company.Reasons[0]
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n%s%s", index+1, company.Name, company.Description, explanation))
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, company.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingCompany)
	return ctx.Send("Подходящие компании подобраны по вашим интересам:\n"+strings.Join(lines, "\n")+"\n\nВыберите компанию:", maxbot.WithKeyboard(keyboard))
}

func showBigDirections(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	directions, err := app.Trajectory.GetCareerDirections(context.Background(), id, dto.GetCareerDirectionsRequest{CompanyID: s.CompanyID})
	if err != nil {
		return ctx.Send("Не удалось получить направления компании.")
	}
	if len(directions) == 0 {
		utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingPathResolution)
		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Сменить работодателя")
		return ctx.Send("У выбранного работодателя нет направлений с полным путём до возможности работодателя. Выберите другого работодателя.", maxbot.WithKeyboard(kb))
	}
	s.DirectionIDs = nil
	s.DirectionNames = nil
	keyboard := model.NewKeyboard()
	lines := make([]string, 0, len(directions))
	for index, direction := range directions {
		s.DirectionIDs = append(s.DirectionIDs, direction.ID)
		s.DirectionNames = append(s.DirectionNames, direction.Name)
		lines = append(lines, fmt.Sprintf("%d. %s — %s", index+1, direction.Name, direction.Description))
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, direction.Name))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingDirection)
	prefix := fmt.Sprintf("В компании «%s» вам подходят следующие карьерные направления.", s.CompanyName)
	if s.Grade == 11 && len(s.SelectedExamIDs) > 0 {
		prefix = fmt.Sprintf("По вашим выбранным ЕГЭ и указанным баллам в компании «%s» вам подходят следующие направления.", s.CompanyName)
	}
	return ctx.Send(prefix+"\n\nВыберите карьерное направление:\n\n"+strings.Join(lines, "\n\n"), maxbot.WithKeyboard(keyboard))
}

func showBigRecommendedExamSets(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	assessment, err := assessCurrentTrajectoryPath(ctx)
	if err != nil {
		return ctx.Send("Не удалось подобрать наборы ЕГЭ.")
	}
	if assessment.Status != dto.TrajectoryPathStatusAvailable {
		return showBigPathResolution(ctx, assessment)
	}
	sets := assessment.ExamSets
	if len(sets) == 0 {
		return showBigPathResolution(ctx, dto.TrajectoryPathAssessmentResponse{Status: dto.TrajectoryPathStatusUnavailable})
	}
	s.ExamSets = nil
	s.ExamSetNames = nil
	keyboard := model.NewKeyboard()
	lines := make([]string, 0, len(sets))
	for index, set := range sets {
		s.ExamSets = append(s.ExamSets, set.ExamSubjectIDs)
		names := make([]string, 0, len(set.Subjects))
		for _, subject := range set.Subjects {
			names = append(names, subject.Name)
		}
		s.ExamSetNames = append(s.ExamSetNames, names)
		lines = append(lines, fmt.Sprintf("%d. %s", index+1, strings.Join(names, ", ")))
		keyboard.AddRow().AddMessage(fmt.Sprintf("%d. Набор %d", index+1, index+1))
	}
	utils.UpdateUserStateStorage(id, models.UserStateBigSurveyWaitingExamSet)
	return ctx.Send(
		fmt.Sprintf("Выберите рекомендуемый набор ЕГЭ. Используем последние доступные правила приёма — %d год:\n\n%s\n\nВведите номер набора или нажмите кнопку.", sets[0].SourceYear, strings.Join(lines, "\n")),
		maxbot.WithKeyboard(keyboard),
	)
}

func continueBigTrajectoryAfterDirection(ctx maxbot.Context) error {
	s := utils.GetSmallSurvey(ctx.Update().UserID)
	if (s.Grade != 11 || len(s.SelectedExamIDs) == 0) && len(s.ActualExamIDs) == 0 {
		return showBigRecommendedExamSets(ctx)
	}
	assessment, err := assessCurrentTrajectoryPath(ctx)
	if err != nil {
		return ctx.Send("Не удалось проверить выбранный набор ЕГЭ.")
	}
	if assessment.Status != dto.TrajectoryPathStatusAvailable {
		return showBigPathResolution(ctx, assessment)
	}
	return showGoalConfirmation(ctx, models.UserStateBigSurveyWaitingGoalConfirmation)
}

func showBigPathResolution(ctx maxbot.Context, assessment dto.TrajectoryPathAssessmentResponse) error {
	return showPathResolution(ctx, models.UserStateBigSurveyWaitingPathResolution, assessment)
}

func sendRoadmap(ctx maxbot.Context, roadmap dto.RoadmapResponse) error {
	steps := make([]string, 0, len(roadmap.Steps))
	for _, step := range roadmap.Steps {
		steps = append(steps, fmt.Sprintf("%d. %s", step.OrderNo, step.Title))
	}
	var summary string
	if roadmap.NextAction == nil {
		summary = "Цель подтверждена. Roadmap сформирован:\n" + strings.Join(steps, "\n")
	} else {
		summary = "Цель подтверждена. Roadmap сформирован:\n" + strings.Join(steps, "\n") + "\n\nСледующее действие: " + roadmap.NextAction.Title
	}
	if err := ctx.Send(summary); err != nil {
		return err
	}
	return startRoadmapScenario(ctx, roadmap)
}
