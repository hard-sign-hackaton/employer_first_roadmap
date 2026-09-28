package handlers

import (
	"context"
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func CreateUser(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.UpdateUserStateStorage(userID, UserStateStart)
	return CallMenu(ctx)
}

// RestartScenario полностью очищает персональный сценарий и возвращает к первому вопросу.
func RestartScenario(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	if err := app.Roadmap.RestartActiveRoadmap(context.Background(), userID); err != nil {
		return ctx.Send("Не удалось очистить сценарий. Попробуйте /restart ещё раз.")
	}
	utils.ResetSmallSurvey(userID)
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

	if userState == UserStateStart {
		kb.AddRow().AddCallBack("Да", "/small_survey").AddCallBack("Нет", "/big_survey")
		return ctx.Send("Вы знаете компанию, в которой хотели бы работать?", maxbot.WithKeyboard(kb))
	}

	return nil
}
