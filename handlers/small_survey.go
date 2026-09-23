package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func CallSmallSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID

	utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingCompanyName)

	return ctx.Send("1. Введите название компании:")
}

func SmallSurveyRelocationConfirm(ctx maxbot.Context) error {
	// TODO: записать статус для релокации
	return proceedToExamsSelection(ctx)
}
func SmallSurveyRelocationDeny(ctx maxbot.Context) error {
	// TODO: записать статус для релокации
	return proceedToExamsSelection(ctx)
}

func proceedToExamsSelection(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingExamSelection)

	kb := model.NewKeyboard()
	kb.AddRow().AddCallBack("Да", "/small_exam_selection_yes").AddCallBack("Нет", "/small_exam_selection_no")
	return ctx.Send("5. Вы уже выбрали предметы на ЕГЭ?", maxbot.WithKeyboard(kb))
}

func SmallSurveyExamSelectionConfirm(ctx maxbot.Context) error {
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Предмет 1")
	kb.AddRow().AddMessage("Предмет 2")
	kb.AddRow().AddMessage("Предмет 3")
	return ctx.Send("Какой из экзаменов вы будете сдавать?", maxbot.WithKeyboard(kb))
}

func SmallSurveyExamSelectionDeny(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.UpdateUserStateStorage(userID, UserStateSurveyCompleted)
	return CallMenu(ctx)
}
