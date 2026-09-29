package handlers

import (
	"context"
	"efr_bot/dto"
	. "efr_bot/models"
	"efr_bot/utils"
	"fmt"
	"strings"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func CallSmallSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.ResetSmallSurvey(userID)
	utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingCompanyName)
	return showSmallCompanies(ctx)
}

func showSmallCompanies(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	companies, err := app.Trajectory.FindCompanies(context.Background(), dto.FindCompaniesRequest{Limit: 10})
	if err != nil {
		return ctx.Send("Не удалось получить компании. Попробуйте позже.")
	}
	if len(companies) == 0 {
		utils.UpdateUserStateStorage(userID, UserStateStart)
		return ctx.Send("Каталог работодателей пока пуст. Работодатель должен сначала заполнить данные через API. Отправьте /start и выберите «Нет», чтобы пройти полный опрос.")
	}
	s := utils.GetSmallSurvey(userID)
	s.CompanyIDs = nil
	kb := model.NewKeyboard()
	lines := make([]string, 0, len(companies))
	for i, c := range companies {
		s.CompanyIDs = append(s.CompanyIDs, c.ID)
		lines = append(lines, fmt.Sprintf("%d. %s — %s", i+1, c.Name, c.Description))
		kb.AddRow().AddMessage(catalogOptionButton(i+1, c.Name))
	}
	kb.AddRow().AddMessage("Не знаю куда хочу")
	return ctx.Send("1. Выберите компанию:\n\n"+strings.Join(lines, "\n\n")+"\n\nНажмите номер компании.", maxbot.WithKeyboard(kb))
}
