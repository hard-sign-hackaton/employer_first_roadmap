package main

import (
	"context"
	"efr_bot/database"
	"efr_bot/handlers"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/max-messenger/maxbot"
)

func init() {
	godotenv.Load()
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.Open(ctx)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Не удалось применить миграции: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Не удалось получить подключение к базе данных: %v", err)
	}
	defer sqlDB.Close()

	// Получение токена бота из переменных окружения
	access_token := os.Getenv("BOT_TOKEN")

	// Создание нового экземпляра бота
	bot, err := maxbot.NewApi(access_token)
	if err != nil {
		log.Fatal(err)
	}

	// ========== Обработка событий ==========
	bot.Handle(maxbot.OnBotStarted, handlers.CreateUser)
	bot.Handle(maxbot.OnMessageCreated, handlers.GlobalMessageListener)

	// ========== Обработка коллбеков ==========
	// Малый опрос
	bot.HandleCallback("/small_survey", handlers.CallSmallSurvey)
	// Большой опрос
	bot.HandleCallback("/big_survey", handlers.CallBigSurvey)

	bot.HandleCallback("/menu", handlers.DemoMenuHandler)
	bot.HandleCallback("/demo", handlers.DemoRequestHandler)
	bot.HandleCallback("/form", handlers.DemoFormHandler)

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
