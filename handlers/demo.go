package handlers

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

const (
	StateNone        = ""             // Форма не запущена
	StateWaitingName = "waiting_name" // Для формы ожидается имя (1 аргумент)
	StateWaitingAge  = "waiting_age"  // ДЛя офрмы ожидается возраст (2 аргумент)
)

var (
	userState = make(map[int64]string)            // Хранилище состояний форм
	userData  = make(map[int64]map[string]string) // Хранилище данных форм
	mu        sync.Mutex
)

func DemoRequestHandler(ctx maxbot.Context) error {
	ctx.Reply("Demo response")
	return DemoMenuHandler(ctx)
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
		// Зaполняем имя и переходим на ввод возраста
		userData[userId]["name"] = text
		userState[userId] = StateWaitingAge

		mu.Unlock()

		ctx.Send("2. Введите возраст")

	case StateWaitingAge:
		// Зaполняем возраст и удаляем данные из временного хранилища
		userData[userId]["age"] = text
		userState[userId] = StateNone

		tempName := userData[userId]["name"]
		tempAge := userData[userId]["age"]

		delete(userState, userId)
		delete(userData, userId)

		mu.Unlock()

		ctx.Send(fmt.Sprintf("Ваше имя: %s\nВаш возраст: %s", tempName, tempAge))
		return DemoMenuHandler(ctx)
	}
	return nil
}

// Меню создает кнопки для вызова коллбеков
func DemoMenuHandler(ctx maxbot.Context) error {
	kb := model.NewKeyboard()
	kb.AddRow().AddCallBack("Демо-запрос", "/demo")
	kb.AddRow().AddCallBack("Заполнить демо-форму", "/form")

	return ctx.Send("Выберите действие", maxbot.WithKeyboard(kb))
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
