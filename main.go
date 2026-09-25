package main

import (
	"context"
	"efr_bot/database"
	"efr_bot/handlers"
	"efr_bot/repositories"
	"efr_bot/services"
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
	if database.DemoSeedEnabled() {
		if err := database.SeedDemoData(db); err != nil {
			log.Fatalf("Не удалось заполнить демонстрационные данные: %v", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Не удалось получить подключение к базе данных: %v", err)
	}
	defer sqlDB.Close()

	profileRepository := repositories.NewGormProfileRepository(db)
	referenceRepository := repositories.NewGormReferenceRepository(db)
	careerRepository := repositories.NewGormCareerRepository(db)
	educationRepository := repositories.NewGormEducationRepository(db)
	roadmapRepository := repositories.NewGormRoadmapRepository(db)
	handlers.Configure(handlers.Services{
		Profile:    services.NewProfileService(profileRepository),
		Reference:  services.NewReferenceService(referenceRepository),
		Trajectory: services.NewTrajectoryService(careerRepository, educationRepository, profileRepository, roadmapRepository),
		Roadmap:    services.NewRoadmapService(roadmapRepository, careerRepository),
		Admission:  services.NewAdmissionService(educationRepository, profileRepository, roadmapRepository, careerRepository),
	})

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
	bot.HandleCallback("/subject_toggle", handlers.SubjectToggle)
	bot.HandleCallback("/subjects_done", handlers.SubjectsDone)
	bot.HandleCallback("/exam_subject_toggle", handlers.ExamSubjectToggle)
	bot.HandleCallback("/exam_subjects_done", handlers.ExamSubjectsDone)
	bot.HandleCallback("/admission_program_toggle", handlers.AdmissionProgramToggle)
	bot.HandleCallback("/admission_plan_done", handlers.AdmissionPlanDone)
	bot.HandleCallback("/admission_plan_reset", handlers.AdmissionPlanReset)

	// Старые кнопки demo-меню тоже возвращают пользователя в актуальный сценарий опроса.
	bot.HandleCallback("/menu", handlers.CallMenu)
	bot.HandleCallback("/demo", handlers.CallMenu)
	bot.HandleCallback("/form", handlers.CallMenu)

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
