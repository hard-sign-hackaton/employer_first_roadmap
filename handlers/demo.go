package handlers

import (
	"fmt"
	"strconv"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func DemoHandler(ctx maxbot.Context) error {
	return ctx.Reply("Demo response")
}

// Доступ к данным пользователя и тексту сообщения
func DemoVerboseEchoHandler(ctx maxbot.Context) error {
	reply := fmt.Sprintf("UserID: %s\nMessageID: %s\nText: %s",
		strconv.FormatInt(ctx.Update().UserID, 10),
		ctx.Update().MessageID,
		ctx.Update().Message.Body.Text,
	)
	return ctx.Reply(reply)
}

func DemoMenuHandler(ctx maxbot.Context) error {
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("/demo")

	return ctx.Send("Menu below", maxbot.WithKeyboard(kb))
}
