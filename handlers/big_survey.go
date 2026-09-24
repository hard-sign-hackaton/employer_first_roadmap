package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/maxbot"
)

func CallBigSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.ResetSmallSurvey(userID)
	utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingGrade)
	return ctx.Send("1. Выберите класс:", maxbot.WithKeyboard(gradeKeyboard()))
}
