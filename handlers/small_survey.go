package handlers

import (
	"context"
	"efr_bot/dto"
	. "efr_bot/models"
	"efr_bot/utils"
	"fmt"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func CallSmallSurvey(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	utils.ResetSmallSurvey(userID)
	utils.UpdateUserStateStorage(userID, UserStateSmallSurveyWaitingCompanyName)
	companies, err := app.Trajectory.FindCompanies(context.Background(), dto.FindCompaniesRequest{Limit: 10})
	if err != nil {
		return ctx.Send("Не удалось получить компании. Попробуйте позже.")
	}
	s := utils.GetSmallSurvey(userID)
	kb := model.NewKeyboard()
	for i, c := range companies {
		s.CompanyIDs = append(s.CompanyIDs, c.ID)
		kb.AddRow().AddMessage(fmt.Sprintf("%d. %s", i+1, c.Name))
	}
	return ctx.Send("1. Выберите компанию:", maxbot.WithKeyboard(kb))
}
