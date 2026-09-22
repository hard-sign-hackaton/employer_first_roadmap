package main

import (
	"efr_bot/handlers"
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

	// Определение обработчиков комманд
	bot.Handle("/demo", handlers.DemoHandler)

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
