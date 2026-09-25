package handlers

import (
	"context"
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func CreateUser(ctx maxbot.Context) error {
	// TODO: создание записи о пользователе в бд
	userID := ctx.Update().UserID

	utils.UpdateUserStateStorage(userID, UserStateStart)

	return CallMenu(ctx)
}

func CallMenu(ctx maxbot.Context) error {
	kb := model.NewKeyboard()
	userID := ctx.Update().UserID
	userState := utils.GetUserState(userID)
	if roadmap, err := app.Roadmap.GetActiveRoadmap(context.Background(), userID); err == nil {
		return showActiveRoadmap(ctx, roadmap)
	}

	switch userState {
	case UserStateStart:
		kb.AddRow().AddCallBack("Да", "/small_survey").AddCallBack("Нет", "/big_survey")
		return ctx.Send("Вы знаете компанию, в которой хотели бы работать?", maxbot.WithKeyboard(kb))
	case UserStateSurveyCompleted:
		// TODO: получать список направлений из бд
		kb.AddRow().AddCallBack("Работа 1", "/")
		kb.AddRow().AddCallBack("Работа 2", "/")
		kb.AddRow().AddCallBack("Работа 3", "/")
		return ctx.Send("Вот возможные профессии для работы в выбраной компании:", maxbot.WithKeyboard(kb))
	}

	return nil
}
