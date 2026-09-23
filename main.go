package main

import (
	"efr_bot/handlers"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/max-messenger/maxbot"
)

func init() {
	godotenv.Load()
}

func main() {
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
	bot.HandleCallback("/small_relocate_yes", handlers.SmallSurveyRelocationConfirm)
	bot.HandleCallback("/small_relocate_no", handlers.SmallSurveyRelocationDeny)
	bot.HandleCallback("/small_exam_selection_yes", handlers.SmallSurveyExamSelectionConfirm)
	bot.HandleCallback("/small_exam_selection_no", handlers.SmallSurveyExamSelectionDeny)
	// Большой опрос
	bot.HandleCallback("/big_survey", handlers.CallBigSurvey)
	bot.HandleCallback("/big_relocate_yes", handlers.BigSurveyRelocationConfirm)
	bot.HandleCallback("/big_relocate_no", handlers.BigSurveyRelocationDeny)
	bot.HandleCallback("/big_exam_selection_yes", handlers.BigSurveyExamSelectionConfirm)
	bot.HandleCallback("/big_exam_selection_no", handlers.BigSurveyExamSelectionDeny)

	bot.HandleCallback("/menu", handlers.DemoMenuHandler)
	bot.HandleCallback("/demo", handlers.DemoRequestHandler)
	bot.HandleCallback("/form", handlers.DemoFormHandler)

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
