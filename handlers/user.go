package handlers

import (
	"context"
	. "efr_bot/models"
	"efr_bot/utils"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func CreateUser(ctx maxbot.Context) error {
	return StartBot(ctx)
}

// StartBot показывает вводную справку и открывает стартовое меню. Его вызывают
// первый вход в бот, а также команды /start и /restart.
func StartBot(ctx maxbot.Context) error {
	utils.UpdateUserStateStorage(ctx.Update().UserID, UserStateStart)
	if err := ctx.Send(welcomeMessage()); err != nil {
		return err
	}
	return CallMenu(ctx)
}

func welcomeMessage() string {
	return "Добро пожаловать!\n\n" +
		"Бот помогает построить путь к работе в интересующей компании: выбрать направление, спланировать ЕГЭ, подобрать вузы и не потерять следующий шаг на всём пути до практики или стажировки.\n\n" +
		"Команды:\n" +
		"/start — открыть стартовое меню;\n" +
		"/roadmap — показать текущую цель, прогресс и продолжить roadmap;\n" +
		"/restart — начать сценарий заново."
}

// RestartScenario полностью очищает персональный сценарий и возвращает к первому вопросу.
func RestartScenario(ctx maxbot.Context) error {
	userID := ctx.Update().UserID
	if err := app.Roadmap.RestartActiveRoadmap(context.Background(), userID); err != nil {
		return ctx.Send("Не удалось очистить сценарий. Попробуйте /restart ещё раз.")
	}
	utils.ResetSmallSurvey(userID)
	return StartBot(ctx)
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
