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

const (
	admissionProgramToggleCallback = "/admission_program_toggle"
	admissionPlanDoneCallback      = "/admission_plan_done"
	admissionPlanResetCallback     = "/admission_plan_reset"
)

func startRoadmapScenario(ctx maxbot.Context, roadmap dto.RoadmapResponse) error {
	s := utils.GetSmallSurvey(ctx.Update().UserID)
	s.RoadmapID = roadmap.ID
	return showActiveRoadmap(ctx, roadmap)
}

func handleRoadmapMessage(ctx maxbot.Context) error {
	id, text := ctx.Update().UserID, strings.TrimSpace(ctx.Update().Message.Body.Text)
	state, s := utils.GetUserState(id), utils.GetSmallSurvey(id)
	switch state {
	case models.UserStateRoadmapOverview:
		switch text {
		case "Продолжить", "Начать обучение", "Открыть возможность работодателя", "Подать заявку в компанию":
			return continueActiveRoadmap(ctx)
		case "Показать весь план":
			return showFullCurrentActiveRoadmap(ctx)
		}
		return resendRoadmapOverview(ctx, "Выберите действие с клавиатуры.")
	case models.UserStateRoadmapExamChoice:
		switch text {
		case "Изменить набор ЕГЭ":
			return showExamSubjectSelection(ctx, models.UserStateRoadmapExamChoiceSubjects)
		case "Набор верный, продолжить":
			return finishRoadmapExamChoice(ctx)
		}
		return resendRoadmapExamChoice(ctx, "Выберите действие с клавиатуры.")
	case models.UserStateRoadmapExamChoiceSubjects:
		return resendExamSubjects(ctx, "Выберите предметы ЕГЭ кнопками ниже и нажмите «Готово».")
	case models.UserStateRoadmapPreparing:
		if text == "Подготовка завершена" {
			completeNextRoadmapStep(ctx)
			return showCurrentActiveRoadmap(ctx)
		}
		return resendRoadmapAction(ctx, "Когда этот этап будет завершён, нажмите «Подготовка завершена».")
	case models.UserStateRoadmapExamReady:
		if text == "ЕГЭ сданы" {
			return showRoadmapExamScores(ctx)
		}
		return resendRoadmapAction(ctx, "Введите баллы только после сдачи ЕГЭ или нажмите «ЕГЭ сданы».")
	case models.UserStateRoadmapExamScores:
		return handleRoadmapExamScore(ctx, text)
	case models.UserStateRoadmapUniversityOptions:
		switch text {
		case "Расширить географию поиска":
			return showRoadmapUniversityOptions(ctx, true)
		case "Начать заново":
			utils.UpdateUserStateStorage(id, models.UserStateStart)
			return CallMenu(ctx)
		case "Готово":
			return saveAdmissionPlan(ctx)
		case "Сбросить выбор":
			s.PlannedAdmissionProgramIDs = nil
			s.PlannedAdmissionProgramLines = nil
			return showRoadmapUniversityOptions(ctx, false)
		}
		index, ok := choice(text, len(s.AdmissionProgramIDs))
		if !ok {
			return resendRoadmapUniversityOptions(ctx, "Выберите вариант с клавиатуры.")
		}
		return toggleAdmissionProgram(ctx, index)
	case models.UserStateRoadmapAdmissionSubmitting:
		if text == "Документы поданы" {
			completeNextRoadmapStep(ctx)
			return showCurrentActiveRoadmap(ctx)
		}
		return resendRoadmapAction(ctx, "Когда документы будут поданы, нажмите «Документы поданы».")
	case models.UserStateRoadmapEnrollmentChoice:
		if text == "Не поступил" {
			if _, err := app.Admission.SaveEnrollmentChoice(context.Background(), id, dto.GetAdmissionPlanRequest{RoadmapID: s.RoadmapID}, dto.SaveEnrollmentChoiceRequest{Status: models.EnrollmentStatusNotEnrolled}); err != nil {
				return ctx.Send("Не удалось сохранить результат приёмной кампании. Попробуйте позже.")
			}
			utils.ResetSmallSurvey(id)
			utils.UpdateUserStateStorage(id, models.UserStateStart)
			if err := ctx.Send("Текущий roadmap сохранён в истории как незавершённый. Начнём новый путь с первого шага: вы сможете выбрать новую компанию, направление и набор ЕГЭ."); err != nil {
				return err
			}
			return CallMenu(ctx)
		}
		index, ok := choice(text, len(s.AdmissionApplicationIDs))
		if !ok {
			return resendEnrollmentChoice(ctx, "Выберите итог приёмной кампании с клавиатуры.")
		}
		return saveEnrollmentChoice(ctx, index)
	case models.UserStateRoadmapLearning:
		if isStartOpportunityAction(text) {
			completeNextRoadmapStep(ctx)
			return showEmployerExperience(ctx)
		}
		if text == "Вернуться к roadmap" {
			return showCurrentActiveRoadmap(ctx)
		}
		return resendLearningProgress(ctx, "Выберите действие с клавиатуры.")
	case models.UserStateRoadmapEmployerExperience:
		if text == "Практика завершена" {
			completeNextRoadmapStep(ctx)
			return showCurrentActiveRoadmap(ctx)
		}
		return resendRoadmapAction(ctx, "Когда возможность работодателя будет завершена, нажмите «Практика завершена».")
	case models.UserStateRoadmapEmployerApplication:
		if text == "Подать заявку" {
			if _, err := app.Roadmap.SubmitEmployerApplication(context.Background(), id, dto.GetRoadmapRequest{RoadmapID: s.RoadmapID}); err != nil {
				return ctx.Send("Не удалось сохранить заявку: " + err.Error())
			}
			completeNextRoadmapStep(ctx)
			return showCurrentActiveRoadmap(ctx)
		}
		return resendRoadmapAction(ctx, "Нажмите «Подать заявку», когда документы для компании готовы.")
	case models.UserStateRoadmapPostAdmission:
		return handleRoadmapPostAdmission(ctx, text)
	}
	return nil
}

func showCurrentActiveRoadmap(ctx maxbot.Context) error {
	roadmap, err := app.Roadmap.GetActiveRoadmap(context.Background(), ctx.Update().UserID)
	if err != nil {
		utils.UpdateUserStateStorage(ctx.Update().UserID, models.UserStateStart)
		return CallMenu(ctx)
	}
	return showActiveRoadmap(ctx, roadmap)
}

func showFullCurrentActiveRoadmap(ctx maxbot.Context) error {
	roadmap, err := app.Roadmap.GetActiveRoadmap(context.Background(), ctx.Update().UserID)
	if err != nil {
		return showCurrentActiveRoadmap(ctx)
	}
	return showFullActiveRoadmap(ctx, roadmap)
}

func showActiveRoadmap(ctx maxbot.Context, roadmap dto.RoadmapResponse) error {
	return sendActiveRoadmap(ctx, roadmap, false)
}

func showFullActiveRoadmap(ctx maxbot.Context, roadmap dto.RoadmapResponse) error {
	return sendActiveRoadmap(ctx, roadmap, true)
}

func sendActiveRoadmap(ctx maxbot.Context, roadmap dto.RoadmapResponse, full bool) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	s.RoadmapID = roadmap.ID
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapOverview)
	steps := make([]string, 0, len(roadmap.Steps))
	completed := 0
	lastCompleted := ""
	for _, step := range roadmap.Steps {
		mark := "○"
		if step.Status == models.RoadmapStepStatusCompleted {
			mark = "✓"
			completed++
			lastCompleted = fmt.Sprintf("✓ Последний завершённый: %d. %s", step.OrderNo, step.Title)
		} else if step.Status == models.RoadmapStepStatusActive {
			mark = "→"
		}
		if full {
			steps = append(steps, fmt.Sprintf("%s %d. %s", mark, step.OrderNo, step.Title))
		}
	}
	text := fmt.Sprintf("Ваш roadmap: выполнено %d из %d шагов.", completed, len(roadmap.Steps))
	if full {
		text += "\n\n" + strings.Join(steps, "\n")
	} else if lastCompleted != "" {
		text += "\n\n" + lastCompleted
	}
	if roadmap.NextAction == nil {
		return ctx.Send(text + "\n\nRoadmap завершён.")
	}
	text += "\n\n→ Следующий шаг: " + roadmap.NextAction.Title
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage(roadmapPrimaryAction(roadmap.NextAction.StepType))
	kb.AddRow().AddMessage("Показать весь план")
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func resendRoadmapOverview(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showCurrentActiveRoadmap(ctx)
}

func resendRoadmapExamChoice(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showRoadmapExamChoice(ctx)
}

func resendRoadmapUniversityOptions(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showRoadmapUniversityOptions(ctx, utils.GetSmallSurvey(ctx.Update().UserID).AdmissionSearchAllRegions)
}

func resendEnrollmentChoice(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showEnrollmentChoice(ctx)
}

func resendLearningProgress(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return showLearningProgress(ctx)
}

func resendRoadmapAction(ctx maxbot.Context, message string) error {
	if err := ctx.Send(message); err != nil {
		return err
	}
	return continueActiveRoadmap(ctx)
}

func continueActiveRoadmap(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	roadmap, err := app.Roadmap.GetActiveRoadmap(context.Background(), id)
	if err != nil || roadmap.NextAction == nil {
		return showCurrentActiveRoadmap(ctx)
	}
	s := utils.GetSmallSurvey(id)
	s.RoadmapID = roadmap.ID
	switch roadmap.NextAction.StepType {
	case models.RoadmapStepTypeChooseOrConfirmExams:
		if err := loadRoadmapExamSubjects(ctx); err != nil {
			return ctx.Send("Не удалось получить сохранённый набор ЕГЭ.")
		}
		return showRoadmapExamChoice(ctx)
	case models.RoadmapStepTypePrepareForExams:
		utils.UpdateUserStateStorage(id, models.UserStateRoadmapPreparing)
		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Подготовка завершена")
		return ctx.Send("Этап подготовки к ЕГЭ. Составьте план, решайте пробные варианты и разберите ошибки.\n\nОтметьте шаг только после завершения подготовки.", maxbot.WithKeyboard(kb))
	case models.RoadmapStepTypePassExams:
		if err := loadRoadmapExamSubjects(ctx); err != nil {
			return ctx.Send("Не удалось получить сохранённый набор ЕГЭ.")
		}
		utils.UpdateUserStateStorage(id, models.UserStateRoadmapExamReady)
		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("ЕГЭ сданы")
		return ctx.Send("Фактические баллы вводятся только после сдачи ЕГЭ. Когда результаты будут известны, нажмите «ЕГЭ сданы».", maxbot.WithKeyboard(kb))
	case models.RoadmapStepTypeChooseUniversity:
		return showRoadmapUniversityOptions(ctx, false)
	case models.RoadmapStepTypeSubmitAdmissionDocuments:
		utils.UpdateUserStateStorage(id, models.UserStateRoadmapAdmissionSubmitting)
		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Документы поданы")
		return ctx.Send("План подачи сохранён. Подайте документы в выбранные вузы и отметьте этот шаг после подачи.", maxbot.WithKeyboard(kb))
	case models.RoadmapStepTypeConfirmEnrollment:
		return showEnrollmentChoice(ctx)
	case models.RoadmapStepTypeLearnAtUniversity:
		if roadmap.EnrollmentChoice == nil || roadmap.EnrollmentChoice.Status != models.EnrollmentStatusChosen {
			return resendEnrollmentChoice(ctx, "Итог приёмной кампании не подтверждён. Вернитесь к выбору результата поступления.")
		}
		return showLearningProgress(ctx)
	case models.RoadmapStepTypeEmployerExperience:
		return showEmployerExperience(ctx)
	case models.RoadmapStepTypeApplyToEmployer:
		return showEmployerApplication(ctx)
	default:
		return ctx.Send("Для этого шага пока нет интерактивного действия.")
	}
}

func roadmapPrimaryAction(stepType string) string {
	switch stepType {
	case models.RoadmapStepTypeLearnAtUniversity:
		return "Начать обучение"
	case models.RoadmapStepTypeEmployerExperience:
		return "Открыть возможность работодателя"
	case models.RoadmapStepTypeApplyToEmployer:
		return "Подать заявку в компанию"
	default:
		return "Продолжить"
	}
}

func loadRoadmapExamSubjects(ctx maxbot.Context) error {
	subjects, err := app.Profile.GetUserSubjects(context.Background(), ctx.Update().UserID)
	if err != nil {
		return err
	}
	s := utils.GetSmallSurvey(ctx.Update().UserID)
	s.PlannedExamIDs, s.PlannedExamNames = nil, nil
	s.SelectedExamIDs, s.SelectedExamNames = nil, nil
	for _, subject := range subjects {
		switch subject.Status {
		case models.SubjectStatusPlanned:
			s.PlannedExamIDs = append(s.PlannedExamIDs, subject.ExamSubjectID)
			s.PlannedExamNames = append(s.PlannedExamNames, subject.Name)
		case models.SubjectStatusSelected, models.SubjectStatusPassed:
			s.SelectedExamIDs = append(s.SelectedExamIDs, subject.ExamSubjectID)
			s.SelectedExamNames = append(s.SelectedExamNames, subject.Name)
		}
	}
	return nil
}

func showRoadmapExamChoice(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapExamChoice)
	examNames := s.SelectedExamNames
	if len(examNames) == 0 {
		examNames = s.PlannedExamNames
	}
	setLabel := strings.Join(examNames, ", ")
	if setLabel == "" {
		setLabel = "не выбран"
	}
	text := "Этап 1. Перед экзаменом.\n\n" +
		"Ваш набор предметов ЕГЭ: " + setLabel + "\n\n" +
		"К экзаменам важно начать готовиться заранее:\n" +
		"• составьте план подготовки и распределите темы по неделям;\n" +
		"• решайте пробные варианты и демоверсии;\n" +
		"• разбирайте ошибки с репетитором или на курсах.\n\n" +
		"Что делаем с набором ЕГЭ?"
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Изменить набор ЕГЭ")
	kb.AddRow().AddMessage("Набор верный, продолжить")
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func finishRoadmapExamChoice(ctx maxbot.Context) error {
	completeNextRoadmapStep(ctx)
	return showCurrentActiveRoadmap(ctx)
}

func roadmapExamSubjects(s *utils.SmallSurveyData) ([]int64, []string) {
	if len(s.SelectedExamIDs) > 0 {
		return s.SelectedExamIDs, s.SelectedExamNames
	}
	return s.PlannedExamIDs, s.PlannedExamNames
}

func showRoadmapExamScores(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	ids, names := roadmapExamSubjects(s)
	if len(ids) == 0 {
		return ctx.Send("Не удалось определить набор предметов ЕГЭ. Начните заново командой /start.")
	}
	s.ActualExamIDs = append([]int64(nil), ids...)
	s.ActualExamNames = append([]string(nil), names...)
	s.ActualExamScores = make(map[int64]int16)
	s.ExamScoreStep = 0
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapExamScores)
	text, kb := actualExamScoreQuestion(s)
	return ctx.Send("Этап 2. После экзаменов.\n\nЭкзамены сданы — запишем фактические баллы по каждому предмету отдельно.\n\n"+text, maxbot.WithKeyboard(kb))
}

func actualExamScoreQuestion(s *utils.SmallSurveyData) (string, *model.Keyboard) {
	text := fmt.Sprintf("Укажите фактический балл по предмету «%s» (0–100):", s.ActualExamNames[s.ExamScoreStep])
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Пропустить")
	return text, kb
}

func handleRoadmapExamScore(ctx maxbot.Context, text string) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	if s.ExamScoreStep >= len(s.ActualExamIDs) {
		return ctx.Send("Баллы уже записаны. Начните заново командой /start.")
	}
	if !strings.EqualFold(text, "Пропустить") {
		score, err := strconv.ParseInt(text, 10, 16)
		if err != nil || score < 0 || score > 100 {
			if err := ctx.Send("Введите число от 0 до 100 или нажмите «Пропустить»."); err != nil {
				return err
			}
			question, kb := actualExamScoreQuestion(s)
			return ctx.Send(question, maxbot.WithKeyboard(kb))
		}
		s.ActualExamScores[s.ActualExamIDs[s.ExamScoreStep]] = int16(score)
	}
	s.ExamScoreStep++
	if s.ExamScoreStep < len(s.ActualExamIDs) {
		text, kb := actualExamScoreQuestion(s)
		return ctx.Send(text, maxbot.WithKeyboard(kb))
	}
	results := make([]dto.ExamResultInput, 0, len(s.ActualExamScores))
	for subjectID, score := range s.ActualExamScores {
		results = append(results, dto.ExamResultInput{ExamSubjectID: subjectID, ActualScore: score})
	}
	if len(results) > 0 {
		if _, err := app.Profile.SaveExamResults(context.Background(), id, dto.SaveExamResultsRequest{Results: results}); err != nil {
			return ctx.Send("Не удалось сохранить результаты ЕГЭ. Попробуйте позже.")
		}
	}
	completeNextRoadmapStep(ctx)
	return showCurrentActiveRoadmap(ctx)
}

func showRoadmapUniversityOptions(ctx maxbot.Context, expandGeography bool) error {
	return renderRoadmapUniversityOptions(ctx, expandGeography, false)
}

func editRoadmapUniversityOptions(ctx maxbot.Context, expandGeography bool) error {
	s := utils.GetSmallSurvey(ctx.Update().UserID)
	if len(s.AdmissionProgramIDs) == 0 {
		return ctx.Answer("Список программ устарел. Откройте roadmap ещё раз.")
	}
	text, keyboard := admissionProgramsQuestion(s)
	return ctx.Edit(text, maxbot.WithKeyboard(keyboard))
}

func renderRoadmapUniversityOptions(ctx maxbot.Context, expandGeography, edit bool) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	s.AdmissionProgramIDs = nil
	s.AdmissionOptionLines = nil
	s.AdmissionOptionLabels = nil
	if s.RoadmapID <= 0 {
		return ctx.Send("Не удалось определить roadmap. Начните заново командой /start.")
	}
	profile, profileErr := app.Profile.GetProfile(context.Background(), id)
	searchAllRegions := expandGeography || (profileErr == nil && profile.WillingToRelocate)
	s.AdmissionSearchAllRegions = searchAllRegions
	options, err := app.Admission.FindEducationOptions(context.Background(), id, dto.FindEducationOptionsRequest{RoadmapID: s.RoadmapID, ExpandGeography: searchAllRegions})
	if err != nil || len(options) == 0 {
		return handleNoUniversityOptions(ctx, searchAllRegions)
	}
	for index, option := range options {
		s.AdmissionProgramIDs = append(s.AdmissionProgramIDs, option.EducationProgramID)
		line := fmt.Sprintf("%d. %s — «%s»\n   Регион: %s; правила ЕГЭ: %d; мин. сумма: %s; бюджет: %s; платно: %s%s",
			index+1, option.UniversityName, option.ProgramName, option.UniversityRegion,
			option.RulesSourceYear, scoreOrDash(option.MinimumTotalScore), scoreOrDash(option.BudgetPassingScore), scoreOrDash(option.PaidPassingScore), passingScoreSourceLabel(option.PassingScoreSourceYear))
		s.AdmissionOptionLines = append(s.AdmissionOptionLines, line)
		s.AdmissionOptionLabels = append(s.AdmissionOptionLabels, fmt.Sprintf("%d. %s", index+1, option.UniversityName))
	}
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapUniversityOptions)
	text, kb := admissionProgramsQuestion(s)
	if edit {
		return ctx.Edit(text, maxbot.WithKeyboard(kb))
	}
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func admissionProgramsQuestion(s *utils.SmallSurveyData) (string, *model.Keyboard) {
	kb := model.NewKeyboard()
	for index, programID := range s.AdmissionProgramIDs {
		label := s.AdmissionOptionLabels[index]
		if admissionProgramSelected(s, programID) {
			label = "✓ " + label
		}
		kb.AddRow().AddCallBack(label, fmt.Sprintf("%s:%d", admissionProgramToggleCallback, programID))
	}
	kb.AddRow().AddCallBack("Готово", admissionPlanDoneCallback)
	kb.AddRow().AddCallBack("Сбросить выбор", admissionPlanResetCallback)
	text := "Этап 3. План поступления.\n\nПодобраны варианты под ваши результаты ЕГЭ и карьерное направление:\n\n" + strings.Join(s.AdmissionOptionLines, "\n\n") + "\n\nВыберите несколько программ, затем нажмите «Готово». Выбрано: " + strconv.Itoa(len(s.PlannedAdmissionProgramIDs))
	return text, kb
}

func handleNoUniversityOptions(ctx maxbot.Context, expandGeography bool) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapUniversityOptions)
	s.AdmissionProgramIDs = nil
	s.AdmissionOptionLines = nil
	s.AdmissionOptionLabels = nil
	profile, profileErr := app.Profile.GetProfile(context.Background(), id)
	willingToRelocate := profileErr == nil && profile.WillingToRelocate
	diagnosis, diagnosisErr := app.Admission.DiagnoseEducationOptions(context.Background(), id, dto.FindEducationOptionsRequest{RoadmapID: s.RoadmapID, ExpandGeography: expandGeography})
	if diagnosisErr != nil {
		return ctx.Send("Не удалось определить причину, по которой не нашлись вузы. Попробуйте позже.")
	}
	if diagnosis.Reason == "region_restriction" && !willingToRelocate && !expandGeography {
		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Расширить географию поиска")
		kb.AddRow().AddMessage("Начать заново")
		return ctx.Send("По вашим ЕГЭ и баллам подходящие программы есть, но в регионе «"+profile.Region.Name+"» их нет в демо-каталоге.\n\nВы указали, что не готовы к переезду. Хотите посмотреть варианты в других регионах?", maxbot.WithKeyboard(kb))
	}
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Начать заново")
	return ctx.Send(noUniversityOptionsMessage(diagnosis), maxbot.WithKeyboard(kb))
}

func noUniversityOptionsMessage(diagnosis dto.EducationOptionsDiagnosisResponse) string {
	switch diagnosis.Reason {
	case "missing_exam_results":
		return "Не найдены фактические баллы ЕГЭ. Укажите балл хотя бы по одному предмету и попробуйте снова."
	case "no_catalog_data":
		return "Для выбранного карьерного направления в демо-каталоге пока нет образовательных программ с опубликованными правилами ЕГЭ."
	case "exam_subjects_mismatch":
		return "В базе есть программы для выбранного направления, но ни одна их комбинация ЕГЭ не совпадает с сохранёнными предметами. Пересмотрите набор ЕГЭ или направление."
	case "minimum_scores_not_met":
		return "Ваш набор ЕГЭ подходит, но фактические баллы ниже минимальных требований доступных программ. Проверьте введённые баллы или пересмотрите траекторию."
	case "region_restriction":
		return "В вашем регионе подходящих программ нет. В других регионах варианты есть, но поиск по всей стране сейчас не включён."
	default:
		return "Подходящих вариантов в каталоге не найдено. Попробуйте позже или пересмотрите карьерное направление."
	}
}

func scoreOrDash(score *int16) string {
	if score == nil {
		return "—"
	}
	return strconv.FormatInt(int64(*score), 10)
}

func passingScoreSourceLabel(year *int16) string {
	if year == nil {
		return ""
	}
	return fmt.Sprintf(" (проходные баллы за %d)", *year)
}

func toggleAdmissionProgram(ctx maxbot.Context, index int) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	if index < 0 || index >= len(s.AdmissionProgramIDs) {
		return resendRoadmapUniversityOptions(ctx, "Не удалось определить выбранный вариант.")
	}
	programID := s.AdmissionProgramIDs[index]
	toggleAdmissionProgramSelection(s, programID, index)
	return showRoadmapUniversityOptions(ctx, s.AdmissionSearchAllRegions)
}

func toggleAdmissionProgramSelection(s *utils.SmallSurveyData, programID int64, index int) {
	for selectedIndex, selectedID := range s.PlannedAdmissionProgramIDs {
		if selectedID == programID {
			s.PlannedAdmissionProgramIDs = append(s.PlannedAdmissionProgramIDs[:selectedIndex], s.PlannedAdmissionProgramIDs[selectedIndex+1:]...)
			s.PlannedAdmissionProgramLines = append(s.PlannedAdmissionProgramLines[:selectedIndex], s.PlannedAdmissionProgramLines[selectedIndex+1:]...)
			return
		}
	}
	s.PlannedAdmissionProgramIDs = append(s.PlannedAdmissionProgramIDs, programID)
	s.PlannedAdmissionProgramLines = append(s.PlannedAdmissionProgramLines, s.AdmissionOptionLines[index])
}

// AdmissionProgramToggle меняет выбор программы и редактирует исходное сообщение.
func AdmissionProgramToggle(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	if utils.GetUserState(id) != models.UserStateRoadmapUniversityOptions {
		return ctx.Answer("Выбор программ уже завершён.")
	}
	payload := ctx.Update().GetCallbackPayload()
	programID, err := strconv.ParseInt(payload.Param, 10, 64)
	if err != nil {
		return ctx.Answer("Не удалось определить программу.")
	}
	s := utils.GetSmallSurvey(id)
	index := -1
	for i, id := range s.AdmissionProgramIDs {
		if id == programID {
			index = i
			break
		}
	}
	if index < 0 {
		return ctx.Answer("Этот список программ устарел. Откройте roadmap ещё раз.")
	}
	toggleAdmissionProgramSelection(s, programID, index)
	programLabel := s.AdmissionOptionLabels[index]
	if admissionProgramSelected(s, programID) {
		_ = ctx.Answer("✓ Выбрано: " + programLabel)
	} else {
		_ = ctx.Answer("Снято: " + programLabel)
	}
	return editRoadmapUniversityOptions(ctx, s.AdmissionSearchAllRegions)
}

func AdmissionPlanReset(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	if utils.GetUserState(id) != models.UserStateRoadmapUniversityOptions {
		return ctx.Answer("Выбор программ уже завершён.")
	}
	s := utils.GetSmallSurvey(id)
	s.PlannedAdmissionProgramIDs = nil
	s.PlannedAdmissionProgramLines = nil
	_ = ctx.Answer("Выбор очищен")
	return editRoadmapUniversityOptions(ctx, s.AdmissionSearchAllRegions)
}

func AdmissionPlanDone(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	if utils.GetUserState(id) != models.UserStateRoadmapUniversityOptions {
		return ctx.Answer("Выбор программ уже завершён.")
	}
	if len(utils.GetSmallSurvey(id).PlannedAdmissionProgramIDs) == 0 {
		if err := ctx.Send("Выберите хотя бы одну программу."); err != nil {
			return err
		}
		return editRoadmapUniversityOptions(ctx, utils.GetSmallSurvey(id).AdmissionSearchAllRegions)
	}
	_ = ctx.Answer("План поступления сохраняется")
	return saveAdmissionPlan(ctx)
}

func admissionProgramSelected(s *utils.SmallSurveyData, programID int64) bool {
	for _, selectedID := range s.PlannedAdmissionProgramIDs {
		if selectedID == programID {
			return true
		}
	}
	return false
}

func saveAdmissionPlan(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	if len(s.PlannedAdmissionProgramIDs) == 0 {
		return resendRoadmapUniversityOptions(ctx, "Выберите хотя бы одну программу для плана подачи.")
	}
	inputs := make([]dto.AdmissionPlanItemInput, 0, len(s.PlannedAdmissionProgramIDs))
	for _, programID := range s.PlannedAdmissionProgramIDs {
		inputs = append(inputs, dto.AdmissionPlanItemInput{EducationProgramID: programID})
	}
	applications, err := app.Admission.SaveAdmissionPlan(context.Background(), id,
		dto.GetAdmissionPlanRequest{RoadmapID: s.RoadmapID},
		dto.SaveAdmissionPlanRequest{Applications: inputs})
	if err != nil || len(applications) == 0 {
		return ctx.Send("Не удалось сохранить план поступления: проверьте лимиты вузов и программ.")
	}
	completeNextRoadmapStep(ctx)
	if err := ctx.Send(fmt.Sprintf("План подачи сохранён: %d программ(ы). Итоговое зачисление вы укажете после завершения приёмной кампании.", len(applications))); err != nil {
		return err
	}
	return showCurrentActiveRoadmap(ctx)
}

func showEnrollmentChoice(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	roadmap, err := app.Roadmap.GetRoadmap(context.Background(), id, dto.GetRoadmapRequest{RoadmapID: s.RoadmapID})
	if err != nil {
		return ctx.Send("Не удалось получить текущий roadmap.")
	}
	if roadmap.EnrollmentChoice != nil && roadmap.EnrollmentChoice.Status == models.EnrollmentStatusChosen {
		utils.UpdateUserStateStorage(id, models.UserStateRoadmapOverview)
		return ctx.Send("Итог зачисления уже сохранён: " + roadmap.EnrollmentChoice.UniversityName + " — «" + roadmap.EnrollmentChoice.EducationProgramName + "». Следующий этап roadmap — обучение.")
	}
	applications, err := app.Admission.GetAdmissionPlan(context.Background(), id, dto.GetAdmissionPlanRequest{RoadmapID: s.RoadmapID})
	if err != nil {
		return ctx.Send("Не удалось получить сохранённый план поступления.")
	}
	if len(applications) == 0 {
		return ctx.Send("Сначала сформируйте и сохраните план подачи документов.")
	}
	s.AdmissionApplicationIDs = nil
	s.AdmissionApplicationLines = nil
	kb := model.NewKeyboard()
	lines := make([]string, 0, len(applications))
	for index, application := range applications {
		s.AdmissionApplicationIDs = append(s.AdmissionApplicationIDs, application.ID)
		line := fmt.Sprintf("%d. %s — «%s»", index+1, application.UniversityName, application.ProgramName)
		s.AdmissionApplicationLines = append(s.AdmissionApplicationLines, line)
		lines = append(lines, line)
		kb.AddRow().AddMessage(line)
	}
	kb.AddRow().AddMessage("Не поступил")
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapEnrollmentChoice)
	return ctx.Send("Этап 4. Результаты приёмной кампании.\n\nПриёмная кампания завершена. Укажите итоговый вуз и программу:\n\n"+strings.Join(lines, "\n"), maxbot.WithKeyboard(kb))
}

func saveEnrollmentChoice(ctx maxbot.Context, index int) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	if index < 0 || index >= len(s.AdmissionApplicationIDs) {
		return resendEnrollmentChoice(ctx, "Не удалось определить выбранную заявку.")
	}
	applicationID := s.AdmissionApplicationIDs[index]
	if _, err := app.Admission.SaveEnrollmentChoice(context.Background(), id,
		dto.GetAdmissionPlanRequest{RoadmapID: s.RoadmapID},
		dto.SaveEnrollmentChoiceRequest{Status: models.EnrollmentStatusChosen, AdmissionApplicationID: &applicationID}); err != nil {
		return ctx.Send("Не удалось сохранить итог зачисления. Попробуйте позже.")
	}
	completeNextRoadmapStep(ctx)
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapOverview)
	if err := ctx.Send("Итог зачисления сохранён. Следующий шаг — обучение в выбранном вузе."); err != nil {
		return err
	}
	return showCurrentActiveRoadmap(ctx)
}

func showLearningProgress(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	roadmap, err := app.Roadmap.GetRoadmap(context.Background(), id, dto.GetRoadmapRequest{RoadmapID: s.RoadmapID})
	if err != nil || roadmap.EnrollmentChoice == nil || roadmap.EnrollmentChoice.Status != models.EnrollmentStatusChosen {
		return ctx.Send("Для учебного этапа нужно сначала подтвердить зачисление.")
	}
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapLearning)
	choice := roadmap.EnrollmentChoice
	text := fmt.Sprintf("Учебный этап: %s — «%s».", choice.UniversityName, choice.EducationProgramName)
	kb := model.NewKeyboard()
	if opportunity, err := app.Roadmap.GetEmployerOpportunity(context.Background(), id, dto.GetRoadmapRequest{RoadmapID: s.RoadmapID}); err == nil {
		text += fmt.Sprintf("\n\nВ каталоге есть возможность работодателя: %s. Она доступна с %d курса.", opportunity.Name, opportunity.MinStudyYear)
		kb.AddRow().AddMessage(startOpportunityAction(opportunity.Type))
	} else {
		text += "\n\nДля выбранной компании, направления и региона вуза в каталоге пока нет возможности работодателя. Учебный этап остаётся активным до появления данных."
		kb.AddRow().AddMessage("Вернуться к roadmap")
	}
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func startOpportunityAction(opportunityType string) string {
	switch opportunityType {
	case models.OpportunityTypeInternship:
		return "Начать стажировку"
	case models.OpportunityTypePractice:
		return "Начать практику"
	case models.OpportunityTypeProject:
		return "Начать проект"
	case models.OpportunityTypeHackathon:
		return "Начать хакатон"
	case models.OpportunityTypeTargetedTraining:
		return "Начать целевое обучение"
	default:
		return "Начать возможность работодателя"
	}
}

func isStartOpportunityAction(text string) bool {
	for _, opportunityType := range []string{
		models.OpportunityTypeInternship,
		models.OpportunityTypePractice,
		models.OpportunityTypeProject,
		models.OpportunityTypeHackathon,
		models.OpportunityTypeTargetedTraining,
		"",
	} {
		if text == startOpportunityAction(opportunityType) {
			return true
		}
	}
	return false
}

func showEmployerExperience(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	opportunity, err := app.Roadmap.GetEmployerOpportunity(context.Background(), id, dto.GetRoadmapRequest{RoadmapID: s.RoadmapID})
	if err != nil {
		return ctx.Send("Для текущего курса нет подтверждённой возможности работодателя.")
	}
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapEmployerExperience)
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Практика завершена")
	return ctx.Send(fmt.Sprintf("Возможность работодателя: %s\n\n%s", opportunity.Name, opportunity.Description), maxbot.WithKeyboard(kb))
}

func showEmployerApplication(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapEmployerApplication)
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Подать заявку")
	return ctx.Send("Практический этап завершён. Подготовьте документы и подтвердите подачу заявки в компанию.", maxbot.WithKeyboard(kb))
}

func handleRoadmapPostAdmission(ctx maxbot.Context, text string) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	switch text {
	case "Сменить вуз":
		utils.UpdateUserStateStorage(id, models.UserStateRoadmapUniversityOptions)
		return showRoadmapUniversityOptions(ctx, false)
	case "Дополнительные материалы":
		return ctx.Send("Раздел «Дополнительные материалы» в разработке — он будет наполняться по мере накопления данных по направлению «" + s.CareerDirectionName + "». Пока рекомендуем изучить учебный план и программу курса вашего направления.")
	case "Стажировки и вакансии":
		opportunity, err := app.Roadmap.GetEmployerOpportunity(context.Background(), id, dto.GetRoadmapRequest{RoadmapID: s.RoadmapID})
		if err != nil {
			return ctx.Send("Возможности работодателя появятся после подтверждения зачисления и накопления знаний по направлению. Загляните позже.")
		}
		completeNextRoadmapStep(ctx)
		message := fmt.Sprintf("Возможность работодателя:\n%s\n\n%s", opportunity.Name, opportunity.Description)
		if opportunity.URL != "" {
			message += "\n\nПодробнее: " + opportunity.URL
		}
		return ctx.Send(message)
	}
	return ctx.Send("Выберите действие с клавиатуры.")
}

func completeNextRoadmapStep(ctx maxbot.Context) {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	if s.RoadmapID <= 0 {
		return
	}
	roadmap, err := app.Roadmap.GetRoadmap(context.Background(), id, dto.GetRoadmapRequest{RoadmapID: s.RoadmapID})
	if err != nil || roadmap.NextAction == nil || roadmap.NextAction.Status != models.RoadmapStepStatusActive {
		return
	}
	_, _ = app.Roadmap.UpdateRoadmapStep(context.Background(), id,
		dto.GetRoadmapStepRequest{RoadmapID: s.RoadmapID, StepID: roadmap.NextAction.ID},
		dto.UpdateRoadmapStepRequest{Status: models.RoadmapStepStatusCompleted})
}
