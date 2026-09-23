package handlers

import (
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func CallBigSurvey(ctx maxbot.Context) error {

	userID := ctx.Update().UserID

	utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingGrade)

	return ctx.Send("1. В каком вы классе?")
}

func BigSurveyRelocationConfirm(ctx maxbot.Context) error {
	// TODO: записать статус для релокации
	return proceedToExamSelection(ctx)
}
func BigSurveyRelocationDeny(ctx maxbot.Context) error {
	// TODO: записать статус для релокации
	return proceedToExamSelection(ctx)

}

func proceedToExamSelection(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingExamSelection)

	kb := model.NewKeyboard()
	kb.AddRow().AddCallBack("Да", "/big_exam_selection_yes").AddCallBack("Нет", "/big_exam_selection_no")
	return ctx.Send("4. Вы уже выбрали предметы на ЕГЭ?", maxbot.WithKeyboard(kb))
}

func BigSurveyExamSelectionConfirm(ctx maxbot.Context) error {
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("Предмет 1")
	kb.AddRow().AddMessage("Предмет 2")
	kb.AddRow().AddMessage("Предмет 3")
	return ctx.Send("Какой из экзаменов вы будете сдавать?", maxbot.WithKeyboard(kb))
}

func BigSurveyExamSelectionDeny(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.UpdateUserStateStorage(userID, UserStateBigSurveyWaitingInterest)

	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("1")
	kb.AddRow().AddMessage("2")
	kb.AddRow().AddMessage("3")
	return ctx.Send("Что из этого вам было бы интереснее всего делать?", maxbot.WithKeyboard(kb))
}
