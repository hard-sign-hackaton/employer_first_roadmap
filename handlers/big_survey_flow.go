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
			return resendExamScore(ctx, "Введите число от 0 до 100 или нажмите «Пропустить».")
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
		s.SelectedActivityTagIDs = []int64{s.ActivityTagIDs[index]}
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
		if s.Grade == 11 && len(s.SelectedExamIDs) > 0 {
			return showGoalConfirmation(ctx, models.UserStateBigSurveyWaitingGoalConfirmation)
		}
		return showBigRecommendedExamSets(ctx)
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
	tags, err := app.Reference.ListInterestTags(context.Background())
	if err != nil {
		return ctx.Send("Не удалось получить список интересов.")
	}
	s.ActivityTagIDs = nil
	s.ActivityTagNames = nil
	keyboard := model.NewKeyboard()
	actionTags := map[string]bool{
		"Программирование":         true,
		"Аналитика":                true,
		"Инженерия":                true,
		"Коммуникация":             true,
		"Исследования":             true,
		"Управление продуктом":     true,
		"Техническая документация": true,
		"Производство":             true,
		"Медицина":                 true,
		"Экология":                 true,
		"Логистика":                true,
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
	for _, subject := range subjects {
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

	text := "Какие школьные предметы вам интересны? Нажимайте на предметы — выбранные помечаются галочкой. Когда закончите, нажмите «Готово»."
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
	case "Химия":
		return "Химия"
	case "Биология":
		return "Биология"
	case "География":
		return "Экология"
	case "История", "Литература":
		return "Коммуникация"
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
