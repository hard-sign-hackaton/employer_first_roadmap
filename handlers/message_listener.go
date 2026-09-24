package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"
	"fmt"
	"strings"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func GlobalMessageListener(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	if strings.TrimSpace(ctx.Update().Message.Body.Text) == "/start" {
		utils.UpdateUserStateStorage(userID, UserStateStart)
		return CallMenu(ctx)
	}

	userState := utils.GetUserState(userID)
	if strings.HasPrefix(string(userState), "small_survey_") {
		return handleSmallSurveyMessage(ctx)
	}
	if strings.HasPrefix(string(userState), "big_survey_") {
		return handleBigSurveyMessage(ctx)
	}
	if strings.HasPrefix(string(userState), "roadmap_") {
		return handleRoadmapMessage(ctx)
	}

	switch userState {
	case UserStateStart:
		return nil

	// ==================== Small survey ====================
	case UserStateSmallSurveyWaitingCompanyName:
		companyName := ctx.Update().Message.Body.Text

		// TODO: записать название компании в бд

		ctx.Send(fmt.Sprintf("Выбрана компания: %s", companyName))

		utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingGrade)

		return ctx.Send("2. В каком вы классе?")

	case UserStateSmallSurveyWaitingGrade:
		grade := ctx.Update().Message.Body.Text

		// TODO: записать класс пользователя в бд

		ctx.Send(fmt.Sprintf("Выбран класс: %s", grade))

		utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingRegion)

		return ctx.Send("3. Из какого вы региона?")

	case UserStateSmallSurveyWaitingRegion:
		region := ctx.Update().Message.Body.Text

		// TODO: записать регион пользователя в бд

		ctx.Send(fmt.Sprintf("Выбран регион: %s", region))

		utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingRelocation)

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Да").AddMessage("Нет")
		return ctx.Send("4. Вы готовы рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(kb))

	case UserStateSmallSurveyWaitingRelocation:
		// TODO: записать статус для релокации (от ответа)
		// answer := ctx.Update().Message.Body.Text

		utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingExamSelection)

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Да").AddMessage("Нет")
		return ctx.Send("5. Вы уже выбрали предметы на ЕГЭ?", maxbot.WithKeyboard(kb))

	case UserStateSmallSurveyWaitingExamSelection:
		answer := ctx.Update().Message.Body.Text
		if answer == "Нет" {
			utils.UpdateUserStateStorage(userID, UserStateSurveyCompleted)
			return CallMenu(ctx)
		}

		utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingExamSubject)

		// TODO: получить список предметов

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Предмет 1")
		kb.AddRow().AddMessage("Предмет 2")
		kb.AddRow().AddMessage("Предмет 3")
		return ctx.Send("Какой из экзаменов вы будете сдавать?", maxbot.WithKeyboard(kb))

	case UserStateSmallSurveyWaitingExamSubject:
		exam := ctx.Update().Message.Body.Text

		// TODO: записать предмет пользователя в бд

		ctx.Send(fmt.Sprintf("Выбран предмет: %s", exam))

		utils.UpdateUserStateStorage(userID, UserStateSurveyCompleted)

		return CallMenu(ctx)

	// ==================== Big survey ====================
	case UserStateBigSurveyWaitingGrade:
		grade := ctx.Update().Message.Body.Text

		// TODO: записать класс пользователя в бд

		ctx.Send(fmt.Sprintf("Выбран класс: %s", grade))

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingRegion)

		return ctx.Send("2. Из какого вы региона?")

	case UserStateBigSurveyWaitingRegion:
		region := ctx.Update().Message.Body.Text

		// TODO: записать регион пользователя в бд

		ctx.Send(fmt.Sprintf("Выбран регион: %s", region))

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingRelocation)

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Да").AddMessage("Нет")
		return ctx.Send("3. Вы готовы рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(kb))

	case UserStateBigSurveyWaitingRelocation:
		// TODO: записать статус для релокации (от ответа)
		// answer := ctx.Update().Message.Body.Text

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingExamSelection)

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Да").AddMessage("Нет")
		return ctx.Send("4. Вы уже выбрали предметы на ЕГЭ?", maxbot.WithKeyboard(kb))

	case UserStateBigSurveyWaitingExamSelection:
		answer := ctx.Update().Message.Body.Text
		if answer == "Нет" {
			utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingInterest)

			// TODO: получить список активностей/интересов
			kb := model.NewKeyboard()
			kb.AddRow().AddMessage("1")
			kb.AddRow().AddMessage("2")
			kb.AddRow().AddMessage("3")
			return ctx.Send("Что из этого вам было бы интереснее всего делать?", maxbot.WithKeyboard(kb))
		}

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingExamSubject)

		// TODO: получить список предметов

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Предмет 1")
		kb.AddRow().AddMessage("Предмет 2")
		kb.AddRow().AddMessage("Предмет 3")
		return ctx.Send("Какой из экзаменов вы будете сдавать?", maxbot.WithKeyboard(kb))

	case UserStateBigSurveyWaitingExamSubject:
		exam := ctx.Update().Message.Body.Text

		// TODO: записать предмет пользователя в бд

		ctx.Send(fmt.Sprintf("Выбран предмет: %s", exam))

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingExamScore)

		return ctx.Send("Какой балл вы ожидаете получить по этому предмету?")

	case UserStateBigSurveyWaitingExamScore:
		score := ctx.Update().Message.Body.Text

		// TODO: записать ожидаемый балл пользователя в бд

		ctx.Send(fmt.Sprintf("Ожидаемый балл: %s", score))

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingInterest)

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("1")
		kb.AddRow().AddMessage("2")
		kb.AddRow().AddMessage("3")
		return ctx.Send("Что из этого вам было бы интереснее всего делать?", maxbot.WithKeyboard(kb))

	case UserStateBigSurveyWaitingInterest:
		interest := ctx.Update().Message.Body.Text

		// TODO: записать интерес пользователя в бд

		ctx.Send(fmt.Sprintf("Выбрана деятельность: %s", interest))

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingSubjects)

		return ctx.Send("Какие школьные предметы вам интересны больше всего? (Можно несколько)")

	case UserStateBigSurveyWaitingSubjects:
		subjects_string := ctx.Update().Message.Body.Text

		// TODO: записать интерес пользователя в бд

		ctx.Send(fmt.Sprintf("Выбранные предметы: %s", subjects_string))

		// TODO: вывести подобранный набор компаний

		utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingCompany)

		kb := model.NewKeyboard()
		kb.AddRow().AddMessage("Компания 1")
		kb.AddRow().AddMessage("Компания 2")
		kb.AddRow().AddMessage("Компания 3")
		return ctx.Send("Подобраны подходящие компании. В какой из них вы хотели бы работать?", maxbot.WithKeyboard(kb))

	case UserStateBigSurveyWaitingCompany:
		company := ctx.Update().Message.Body.Text

		// TODO: записать компанию пользователя в бд

		ctx.Send(fmt.Sprintf("Выбрана компания: %s", company))

		utils.UpdateUserStateStorage(userID, UserStateSurveyCompleted)

		return CallMenu(ctx)
	}

	return nil
}
