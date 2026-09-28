package handlers

import (
	"context"
	. "efr_bot/models"
	"efr_bot/utils"
	"fmt"
	"strings"

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
		"/feedback — проверить ответ работодателя по заявке;\n" +
		"/restart — начать сценарий заново."
}

func ShowEmployerFeedback(ctx maxbot.Context) error {
	applications, err := app.Roadmap.ListEmployerApplications(context.Background(), ctx.Update().UserID)
	if err != nil {
		return ctx.Send("У вас пока нет заявки работодателю. Она появится после финального шага roadmap.")
	}
	if len(applications) == 0 {
		return ctx.Send("У вас пока нет заявок работодателям.")
	}
	var text strings.Builder
	text.WriteString("Заявки работодателям")
	for index, application := range applications {
		fmt.Fprintf(&text, "\n\n%d. %s", index+1, application.CompanyName)
		if application.OpportunityName != "" {
			fmt.Fprintf(&text, " — %s", application.OpportunityName)
		}
		fmt.Fprintf(&text, "\nСтатус: %s", employerFeedbackStatusLabel(application.Status))
		if application.Message != "" {
			fmt.Fprintf(&text, "\nСообщение: %s", application.Message)
		}
		if application.Contact != "" {
			fmt.Fprintf(&text, "\nКонтакт: %s", application.Contact)
		}
		if application.Status == "submitted" {
			text.WriteString("\nРаботодатель ещё не оставил ответ.")
		}
		if len(application.History) > 1 {
			text.WriteString("\nИстория:")
			for _, feedback := range application.History {
				fmt.Fprintf(&text, "\n• %s — %s", feedback.CreatedAt.Format("02.01.2006 15:04"), employerFeedbackStatusLabel(feedback.Status))
			}
		}
	}
	return ctx.Send(text.String())
}

func employerFeedbackStatusLabel(status string) string {
	switch status {
	case "under_review":
		return "заявка рассматривается"
	case "interview":
		return "приглашение на интервью"
	case "accepted":
		return "заявка принята"
	case "rejected":
		return "заявка отклонена"
	default:
		return "заявка отправлена"
	}
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
