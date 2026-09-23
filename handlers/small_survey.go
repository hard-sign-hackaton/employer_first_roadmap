package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/maxbot"
)

func CallSmallSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID

	utils.UserStateStorageMutex.Lock()
	utils.UserStateStorage[userID] = UserStateSmallSurveyWaitingCompanyName
	utils.UserStateStorageMutex.Unlock()

	return ctx.Send("1. Введите название компании:")
}

func switchToCareerSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.UpdateUserStateStorage(userID, UserStateSurveyCompleted)

	return CallMenu(ctx)
}

func SmallSurveyRelocationConfirm(ctx maxbot.Context) error {
	// TODO: записать статус для релокации
	return switchToCareerSurvey(ctx)
}
func SmallSurveyRelocationDeny(ctx maxbot.Context) error {
	// TODO: записать статус для релокации
	return switchToCareerSurvey(ctx)
}
