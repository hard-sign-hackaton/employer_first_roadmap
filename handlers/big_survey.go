package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/maxbot"
)

func CallBigSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	beginBigSurvey(userID)
	return ctx.Send("1. Выберите класс:", maxbot.WithKeyboard(gradeKeyboard()))
}

// beginBigSurvey очищает состояние малого опроса перед переходом в полный.
func beginBigSurvey(userID int64) {
	utils.ResetSmallSurvey(userID)
	utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingGrade)
}
