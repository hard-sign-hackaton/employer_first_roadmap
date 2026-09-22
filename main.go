package main

import (
	"log"
	"os"

	"github.com/max-messenger/maxbot"
)

func main() {
	// Получение токена бота из переменных окружения
	access_token:= os.Getenv("BOT_TOKEN")

	// Создание нового экземпляра бота
	bot, err := maxbot.NewApi(access_token)
	if err != nil {
		log.Fatal(err)
	}

	// Определение обработчика для команды /start
	bot.Handle("/start", func(ctx maxbot.Context) error {
		return ctx.Reply("Привет! Я хуесос. Чем я могу помочь?")
	})

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
