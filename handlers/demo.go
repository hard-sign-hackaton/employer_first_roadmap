package handlers

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

const (
	StateNone        = "" // Форма не запущена
	StateWaitingName = "waiting_name" // Для формы ожидается имя (1 аргумент)
	StateWaitingAge  = "waiting_age" // ДЛя офрмы ожидается возраст (2 аргумент)
)

var (
	userState = make(map[int64]string)
	userData  = make(map[int64]map[string]string)
	mu        sync.Mutex
)

func DemoHandler(ctx maxbot.Context) error {
	return ctx.Reply("Demo response")
}

// Перехватывает все сообщения
func DemoMessageListenerHandler(ctx maxbot.Context) error {
	userId := ctx.Update().UserID

	mu.Lock()
	state := userState[userId]
	if state == StateNone { // Если форма не запущена, то возвращаем инфу о сообщении
		mu.Unlock()
		reply := fmt.Sprintf("UserID: %s\nMessageID: %s\nText: %s",
			strconv.FormatInt(ctx.Update().UserID, 10),
			ctx.Update().MessageID,
			ctx.Update().Message.Body.Text,
		)
		return ctx.Reply(reply)
	}

	// Если форма запущена, идем по форме
	text := ctx.Update().Message.Body.Text
	switch state {
	case StateWaitingName:
		userData[userId]["name"] = text
		userState[userId] = StateWaitingAge
		mu.Unlock()

		ctx.Send("2. Введите возраст")
	case StateWaitingAge:
		userData[userId]["age"] = text
		userState[userId] = StateNone
		tempName := userData[userId]["name"]
		tempAge := userData[userId]["age"]
		delete(userState, userId)
		delete(userData, userId)
		mu.Unlock()

		ctx.Send(fmt.Sprintf("Ваше имя: %s\nВаш возраст: %s", tempName, tempAge))
	}
	return nil
}

func DemoMenuHandler(ctx maxbot.Context) error {
	kb := model.NewKeyboard()
	kb.AddRow().AddMessage("/demo")
	kb.AddRow().AddMessage("/form")

	return ctx.Send("Menu below", maxbot.WithKeyboard(kb))
}

func DemoFormHandler(ctx maxbot.Context) error {
	userId := ctx.Update().UserID

	mu.Lock()
	userState[userId] = StateWaitingName
	if userData[userId] == nil {
		userData[userId] = make(map[string]string)
	}
	mu.Unlock()

	return ctx.Send("1. Введите имя:")
}
