package handlers

import (
	"fmt"
	"strconv"

	"github.com/max-messenger/maxbot"
)

func DemoHandler(ctx maxbot.Context) error {
	return ctx.Reply("Demo response")
}

// Доступ к данным пользователя и тексту сообщения
func VerboseEchoHandler(ctx maxbot.Context) error {
	reply := fmt.Sprintf("UserID: %s\nMessageID: %s\nText: %s",
					strconv.FormatInt(ctx.Update().UserID, 10),
					ctx.Update().MessageID,
					ctx.Update().Message.Body.Text,
	)
	return ctx.Reply(reply)
}
