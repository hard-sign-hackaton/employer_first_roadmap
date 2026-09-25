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

func startRoadmapScenario(ctx maxbot.Context, roadmap dto.RoadmapResponse) error {
	s := utils.GetSmallSurvey(ctx.Update().UserID)
	s.RoadmapID = roadmap.ID
	return showRoadmapExamChoice(ctx)
}

func handleRoadmapMessage(ctx maxbot.Context) error {
	id, text := ctx.Update().UserID, strings.TrimSpace(ctx.Update().Message.Body.Text)
	state, s := utils.GetUserState(id), utils.GetSmallSurvey(id)
	switch state {
	case models.UserStateRoadmapExamChoice:
		switch text {
		case "Изменить набор ЕГЭ":
			return showExamSubjectSelection(ctx, models.UserStateRoadmapExamChoiceSubjects)
		case "Набор верный, продолжить":
			return finishRoadmapExamChoice(ctx)
		}
		return ctx.Send("Выберите действие с клавиатуры.")
	case models.UserStateRoadmapExamChoiceSubjects:
		return ctx.Send("Выберите предметы ЕГЭ кнопками ниже и нажмите «Готово».")
	case models.UserStateRoadmapExamScores:
		return handleRoadmapExamScore(ctx, text)
	case models.UserStateRoadmapUniversityOptions:
		switch text {
		case "Расширить географию поиска":
			return showRoadmapUniversityOptions(ctx, true)
		case "Начать заново":
			utils.UpdateUserStateStorage(id, models.UserStateStart)
			return CallMenu(ctx)
		}
		index, ok := choice(text, len(s.AdmissionProgramIDs))
		if !ok {
			return ctx.Send("Выберите вариант с клавиатуры.")
		}
		return selectRoadmapProgram(ctx, index)
	case models.UserStateRoadmapUniversityConfirm:
		switch text {
		case "Подтвердить выбор":
			return confirmRoadmapUniversity(ctx)
		case "Выбрать другой вариант":
			return showRoadmapUniversityOptions(ctx, false)
		}
		return ctx.Send("Нажмите «Подтвердить выбор» или «Выбрать другой вариант».")
	case models.UserStateRoadmapPostAdmission:
		return handleRoadmapPostAdmission(ctx, text)
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
	return showRoadmapExamScores(ctx)
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
			return ctx.Send("Введите число от 0 до 100 или нажмите «Пропустить».")
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
	return showRoadmapUniversityOptions(ctx, false)
}

func showRoadmapUniversityOptions(ctx maxbot.Context, expandGeography bool) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	s.AdmissionProgramIDs = nil
	s.AdmissionOptionLines = nil
	if s.RoadmapID <= 0 {
		return ctx.Send("Не удалось определить roadmap. Начните заново командой /start.")
	}
	profile, profileErr := app.Profile.GetProfile(context.Background(), id)
	searchAllRegions := expandGeography || (profileErr == nil && profile.WillingToRelocate)
	options, err := app.Admission.FindEducationOptions(context.Background(), id, dto.FindEducationOptionsRequest{RoadmapID: s.RoadmapID, ExpandGeography: searchAllRegions})
	if err != nil || len(options) == 0 {
		return handleNoUniversityOptions(ctx, searchAllRegions)
	}
	lines := make([]string, 0, len(options))
	kb := model.NewKeyboard()
	for index, option := range options {
		s.AdmissionProgramIDs = append(s.AdmissionProgramIDs, option.EducationProgramID)
		line := fmt.Sprintf("%d. %s — «%s»\n   Регион: %s; правила ЕГЭ: %d; мин. сумма: %s; бюджет: %s; платно: %s%s",
			index+1, option.UniversityName, option.ProgramName, option.UniversityRegion,
			option.RulesSourceYear, scoreOrDash(option.MinimumTotalScore), scoreOrDash(option.BudgetPassingScore), scoreOrDash(option.PaidPassingScore), passingScoreSourceLabel(option.PassingScoreSourceYear))
		lines = append(lines, line)
		s.AdmissionOptionLines = append(s.AdmissionOptionLines, line)
		kb.AddRow().AddMessage(fmt.Sprintf("%d. %s", index+1, option.UniversityName))
	}
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapUniversityOptions)
	text := "Этап 3. Перед поступлением.\n\nПодобраны варианты под ваши результаты ЕГЭ и карьерное направление:\n\n" + strings.Join(lines, "\n\n") + "\n\nВыберите вуз и программу (позже выбор можно изменить):"
	return ctx.Send(text, maxbot.WithKeyboard(kb))
}

func handleNoUniversityOptions(ctx maxbot.Context, expandGeography bool) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapUniversityOptions)
	s.AdmissionProgramIDs = nil
	s.AdmissionOptionLines = nil
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

func selectRoadmapProgram(ctx maxbot.Context, index int) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	if index < 0 || index >= len(s.AdmissionProgramIDs) {
		return ctx.Send("Не удалось определить выбранный вариант.")
	}
	s.SelectedProgramID = s.AdmissionProgramIDs[index]
	s.SelectedProgramLabel = s.AdmissionOptionLines[index]
	return showRoadmapUniversityConfirm(ctx)
}

func showRoadmapUniversityConfirm(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapUniversityConfirm)
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Подтвердить выбор").AddMessage("Выбрать другой вариант")
	return ctx.Send("Вы выбрали:\n"+s.SelectedProgramLabel+"\n\nЗафиксировать этот выбор? Сменить его можно в любой момент.", maxbot.WithKeyboard(kb))
}

func confirmRoadmapUniversity(ctx maxbot.Context) error {
	id := ctx.Update().UserID
	s := utils.GetSmallSurvey(id)
	if s.SelectedProgramID <= 0 {
		return ctx.Send("Не выбран вариант поступления.")
	}
	applications, err := app.Admission.SaveAdmissionPlan(context.Background(), id,
		dto.GetAdmissionPlanRequest{RoadmapID: s.RoadmapID},
		dto.SaveAdmissionPlanRequest{Applications: []dto.AdmissionPlanItemInput{{EducationProgramID: s.SelectedProgramID}}})
	if err != nil || len(applications) == 0 {
		return ctx.Send("Не удалось сохранить план поступления. Попробуйте позже.")
	}
	applicationID := applications[0].ID
	if _, err := app.Admission.SaveEnrollmentChoice(context.Background(), id,
		dto.GetAdmissionPlanRequest{RoadmapID: s.RoadmapID},
		dto.SaveEnrollmentChoiceRequest{Status: models.EnrollmentStatusChosen, AdmissionApplicationID: &applicationID}); err != nil {
		return ctx.Send("Не удалось зафиксировать выбор вуза. Попробуйте позже.")
	}
	completeNextRoadmapStep(ctx)
	completeNextRoadmapStep(ctx)
	utils.UpdateUserStateStorage(id, models.UserStateRoadmapPostAdmission)
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Дополнительные материалы")
	kb.AddRow().AddMessage("Стажировки и вакансии")
	kb.AddRow().AddMessage("Сменить вуз")
	return ctx.Send("Этап 4. После поступления.\n\nВы поступили:\n"+s.SelectedProgramLabel+"\n\nЗдесь доступны дополнительные материалы для учёбы по направлению «"+s.CareerDirectionName+"» и возможности работодателя — стажировки, проекты и вакансии.", maxbot.WithKeyboard(kb))
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
