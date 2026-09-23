package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"
	"fmt"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func GlobalMessageListener(ctx maxbot.Context) error {
	userID := ctx.Update().UserID

	userState := utils.GetUserState(userID)

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
		kb.AddRow().AddCallBack("Да", "/relocate_yes").AddCallBack("Нет", "/relocate_no")
		return ctx.Send("4. Вы готовы рассмотреть обучение в другом регионе?", maxbot.WithKeyboard(kb))

	case UserStateSmallSurveyWaitingExamSelection:
		exam := ctx.Update().Message.Body.Text

		// TODO: записать предмет пользователя в бд

		ctx.Send(fmt.Sprintf("Выбран предмет: %s",  exam))

		utils.UpdateUserStateStorage(userID, UserStateSurveyCompleted)

		return CallMenu(ctx)
	}

	return nil
}
