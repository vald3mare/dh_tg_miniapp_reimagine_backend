package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/handlers"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func seedCatalog(db *gorm.DB) {
	var count int64
	db.Model(&models.CatalogItem{}).Count(&count)
	if count > 0 {
		return
	}

	items := []models.CatalogItem{
		{
			Name:            "Выгул собак",
			Description:     "Полная забота о вашем питомце во время прогулки: от экипировки до игр и обучения",
			FullDescription: "Выгульщик приходит домой сам — с экипировкой или использует вашу.\nОдевает собаку, берёт вкусняшки, пакеты, поводки.\nУчитывает возраст, темперамент, повадки и особенности прогулок.",
			Price:           890,
			ImageURL:        "https://s3.twcstorage.ru/dh-s3-storage/shutterstock_1531627.png",
			Type:            "service",
		},
		{
			Name:            "Зооняня",
			Description:     "Присмотр за питомцем у вас дома или у исполнителя на время вашего отсутствия",
			FullDescription: "Исполнитель остаётся с животным дома.\nКормит по вашему расписанию, выгуливает, отправляет фотоотчёты.\nПодходит для собак, кошек и других животных.",
			Price:           1200,
			ImageURL:        "https://s3.twcstorage.ru/dh-s3-storage/shutterstock_1531627.png",
			Type:            "service",
		},
		{
			Name:            "Кинолог",
			Description:     "Индивидуальные занятия с профессиональным кинологом для вашей собаки",
			FullDescription: "Работаем с послушанием, коррекцией поведения и базовыми командами.\nЗанятия проходят на улице или дома — как удобно вам.\nКинолог адаптирует программу под темперамент и возраст собаки.",
			Price:           1500,
			ImageURL:        "https://s3.twcstorage.ru/dh-s3-storage/shutterstock_1531627.png",
			Type:            "service",
		},
		{
			Name:            "Передержка",
			Description:     "Временное содержание питомца у исполнителя с заботой как дома",
			FullDescription: "Исполнитель принимает животное к себе домой.\nРегулярные прогулки, кормление по вашему рациону, фотоотчёты.\nПодходит на время отпуска или командировки.",
			Price:           700,
			ImageURL:        "https://s3.twcstorage.ru/dh-s3-storage/shutterstock_1531627.png",
			Type:            "service",
		},
		{
			Name:            "Ветеринарная консультация",
			Description:     "Онлайн-консультация ветеринара для оперативных вопросов о здоровье питомца",
			FullDescription: "Ветеринар отвечает на вопросы по питанию, поведению и лечению.\nКонсультация проходит в мессенджере или по видеосвязи.\nПомогаем понять, нужен ли очный визит в клинику.",
			Price:           2000,
			ImageURL:        "https://s3.twcstorage.ru/dh-s3-storage/shutterstock_1531627.png",
			Type:            "service",
		},
	}

	if err := db.Create(&items).Error; err != nil {
		log.Printf("WARN: не удалось добавить seed-данные в каталог: %v", err)
		return
	}
	fmt.Println("Seed: добавлено 5 услуг в каталог")
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// ── База данных ────────────────────────────────────────────────────────────
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", err)
	}

	if err := database.AutoMigrate(
		&models.User{},
		&models.Subscription{},
		&models.CatalogItem{},
		&models.Payment{},
	); err != nil {
		log.Fatalf("AutoMigrate завершился с ошибкой: %v", err)
	}

	seedCatalog(database)

	// ── Роутер ────────────────────────────────────────────────────────────────
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s\" %d %s \"%s\"\n",
			p.ClientIP,
			p.TimeStamp.Format(time.RFC1123),
			p.Method,
			p.Path,
			p.Request.Proto,
			p.StatusCode,
			p.Latency,
			p.Request.UserAgent(),
		)
	}))
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:  []string{"Authorization", "Content-Type"},
		MaxAge:        300,
	}))

	// ── Публичные маршруты ────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/catalog", handlers.GetCatalog(database))

	// Webhook от ЮKassa — публичный, вызывается их серверами
	r.POST("/payment/webhook", handlers.HandlePaymentWebhook(database))

	// ── Защищённые маршруты (требуют Authorization: tma ...) ──────────────────
	auth := middleware.AuthMiddleware(token)
	protected := r.Group("/")
	protected.Use(auth)
	{
		protected.GET("/profile", handlers.GetProfile(database))
		protected.POST("/payment/create", handlers.CreatePayment(database))
	}

	// ── HTTP-сервер с graceful shutdown ───────────────────────────────────────
	srv := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Сервер запущен на :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ждём сигнала завершения (Ctrl+C или kill от systemd/Docker)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Получен сигнал завершения, останавливаем сервер...")

	// Даём 10 секунд на завершение активных запросов
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Принудительное завершение сервера: %v", err)
	}

	log.Println("Сервер остановлен корректно")
}
