package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/maxbot"
)

func CallBigSurvey(ctx maxbot.Context) error {

	userID := ctx.Update().UserID

	utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingGrade)

	return ctx.Send("1. В каком вы классе?")
}
