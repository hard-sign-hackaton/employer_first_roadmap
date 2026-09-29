package main

import (
	"context"
	"efr_bot/database"
	"efr_bot/employerapi"
	"efr_bot/handlers"
	"efr_bot/reminders"
	"efr_bot/repositories"
	"efr_bot/services"
	"log"
	"net/http"
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
	if err := database.SeedReferenceData(db); err != nil {
		log.Fatalf("Не удалось заполнить обязательные справочники: %v", err)
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
	reminderRepository := repositories.NewGormReminderRepository(db)

	// Получение токена бота из переменных окружения
	accessToken := os.Getenv("BOT_TOKEN")
	bot, err := maxbot.NewApi(accessToken)
	if err != nil {
		log.Fatal(err)
	}
	reminderService := reminders.NewService(reminderRepository, reminders.NewMaxSender(bot.Client()))
	handlers.Configure(handlers.Services{
		Profile:    services.NewProfileService(profileRepository),
		Reference:  services.NewReferenceService(referenceRepository),
		Trajectory: services.NewTrajectoryService(careerRepository, educationRepository, profileRepository, roadmapRepository),
		Roadmap:    services.NewRoadmapService(roadmapRepository, careerRepository),
		Admission:  services.NewAdmissionService(educationRepository, profileRepository, roadmapRepository, careerRepository),
		Reminder:   reminderService,
	})

	// Employer API and the bot use the same PostgreSQL catalog. Once an
	// employer saves a catalog, the existing bot services immediately read it.
	apiPort := os.Getenv("EMPLOYER_API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}
	apiServer := &http.Server{
		Addr:              ":" + apiPort,
		Handler:           employerapi.NewHandler(employerapi.NewStore(db), os.Getenv("ADMIN_API_TOKEN"), os.Getenv("EMPLOYER_ALLOWED_ORIGIN")),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Printf("Employer API слушает порт %s", apiPort)
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Employer API остановлен: %v", err)
		}
	}()

	reminderConfig, err := reminders.SchedulerConfigFromEnv()
	if err != nil {
		log.Fatalf("Некорректная конфигурация напоминаний: %v", err)
	}
	if _, err := reminders.StartScheduler(context.Background(), reminderService, reminderConfig); err != nil {
		log.Fatalf("Не удалось запустить планировщик напоминаний: %v", err)
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
	bot.HandleCallback("/open_roadmap", handlers.OpenCurrentRoadmap)

	// Запуск бота и начало мониторинга событий
	log.Println("Бот запускается...")
	bot.Start()
}
