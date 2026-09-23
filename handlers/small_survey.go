package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/maxbot"
)

func CallSmallSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID

	utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingCompanyName)

	return ctx.Send("1. Введите название компании:")
}
