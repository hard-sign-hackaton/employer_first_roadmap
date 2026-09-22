package main

import (
	"efr_bot/handlers"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/max-messenger/maxbot"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Unable to read env")
	}
}

func main() {
	// Получение токена бота из переменных окружения
	access_token := os.Getenv("BOT_TOKEN")

	// Создание нового экземпляра бота
	bot, err := maxbot.NewApi(access_token)
	if err != nil {
		log.Fatal(err)
	}

	// Обработка событий
	bot.Handle(maxbot.OnBotStarted, handlers.DemoMenuHandler)
	bot.Handle(maxbot.OnMessageCreated, handlers.DemoMessageListenerHandler)

	// Обработка коллбеков
	bot.HandleCallback("/menu", handlers.DemoMenuHandler)
	bot.HandleCallback("/demo", handlers.DemoRequestHandler)
	bot.HandleCallback("/form", handlers.DemoFormHandler)

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
