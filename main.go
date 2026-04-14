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
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func seedAchievements(db *gorm.DB) {
	var count int64
	db.Model(&models.Achievement{}).Count(&count)
	if count > 0 {
		return
	}

	achievements := []models.Achievement{
		{Key: "first_order", Name: "Первый шаг", Description: "Выполните первый заказ", IconEmoji: "🐾", ConditionType: "orders_completed", Threshold: 1},
		{Key: "five_orders", Name: "Опытный", Description: "Выполните 5 заказов", IconEmoji: "⭐", ConditionType: "orders_completed", Threshold: 5},
		{Key: "ten_orders", Name: "Профессионал", Description: "Выполните 10 заказов", IconEmoji: "🏆", ConditionType: "orders_completed", Threshold: 10},
		{Key: "verified", Name: "Проверенный", Description: "Пройдите верификацию", IconEmoji: "✅", ConditionType: "manual", Threshold: 0},
	}

	if err := db.Create(&achievements).Error; err != nil {
		log.Printf("WARN: не удалось добавить seed-данные ачивок: %v", err)
		return
	}
	fmt.Println("Seed: добавлено 4 ачивки")
}

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
	// Загружаем .env если он есть (молча игнорируем отсутствие файла)
	if err := godotenv.Load(); err == nil {
		log.Println("Загружен .env файл")
	}

	// ── Дамп переменных окружения при старте (для дебага) ─────────────────────
	mask := func(v string) string {
		if len(v) <= 6 {
			return "***"
		}
		return v[:4] + "..." + v[len(v)-4:]
	}
	log.Printf(`
┌─── ENV CONFIG ────────────────────────────────
│ BOT_TOKEN             = %s
│ ADMIN_TELEGRAM_IDS    = %s
│ EXECUTOR_NOTIFY_CHAT_ID = %s
│ DB_HOST               = %s
│ DB_PORT               = %s
│ DB_USER               = %s
│ DB_PASSWORD           = %s
│ DB_NAME               = %s
│ DB_SSLMODE            = %s
│ YOOKASSA_SHOP_ID      = %s
│ YOOKASSA_SECRET_KEY   = %s
│ YOOKASSA_RETURN_URL   = %s
│ RECEIPT_EMAIL         = %s
│ ORDERS_API_KEY        = %s
│ PORT                  = %s
│ GIN_MODE              = %s
└───────────────────────────────────────────────`,
		mask(os.Getenv("BOT_TOKEN")),
		os.Getenv("ADMIN_TELEGRAM_IDS"),
		os.Getenv("EXECUTOR_NOTIFY_CHAT_ID"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		mask(os.Getenv("DB_PASSWORD")),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
		os.Getenv("YOOKASSA_SHOP_ID"),
		mask(os.Getenv("YOOKASSA_SECRET_KEY")),
		os.Getenv("YOOKASSA_RETURN_URL"),
		os.Getenv("RECEIPT_EMAIL"),
		mask(os.Getenv("ORDERS_API_KEY")),
		os.Getenv("PORT"),
		os.Getenv("GIN_MODE"),
	)

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is not set")
	}

	// Предупреждения для опциональных, но важных переменных
	if os.Getenv("YOOKASSA_SHOP_ID") == "" || os.Getenv("YOOKASSA_SECRET_KEY") == "" {
		log.Println("WARN: YOOKASSA_SHOP_ID или YOOKASSA_SECRET_KEY не заданы — платежи недоступны")
	}
	if os.Getenv("ORDERS_API_KEY") == "" && os.Getenv("ORDERS_AUTH_DISABLED") != "true" {
		log.Println("WARN: ORDERS_API_KEY не задан и ORDERS_AUTH_DISABLED != true — POST /orders будет отклонять все запросы")
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
		&models.Order{},
		&models.Achievement{},
		&models.UserAchievement{},
		&models.ExecutorApplication{},
	); err != nil {
		log.Fatalf("AutoMigrate завершился с ошибкой: %v", err)
	}

	seedCatalog(database)
	seedAchievements(database)

	// ── Роутер ────────────────────────────────────────────────────────────────
	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:       300,
	}))

	// ── Rate limiting: 120 req/min на IP, burst до 20 ────────────────────────
	rl := middleware.NewRateLimiter(rate.Every(time.Minute/120), 20)
	r.Use(rl.Middleware())

	// ── Публичные маршруты ────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/catalog", handlers.GetCatalog(database))
	r.GET("/executor/orders", handlers.GetOpenOrders(database))
	r.POST("/orders", handlers.CreateOrder(database))

	// Вызывается Telegram-ботом после заполнения анкеты исполнителя.
	// Защищён X-API-Key == ORDERS_API_KEY (или ORDERS_AUTH_DISABLED=true).
	r.POST("/bot/application", handlers.SubmitApplication(database))

	// Webhook от ЮKassa — публичный, вызывается их серверами
	r.POST("/payment/webhook", handlers.HandlePaymentWebhook(database))

	auth := middleware.AuthMiddleware(token)

	// ── Маршруты с TMA-авторизацией (/profile создаёт юзера, LoadUser не нужен)
	protected := r.Group("/")
	protected.Use(auth)
	{
		protected.GET("/profile", handlers.GetProfile(database))
	}

	// ── Маршруты с TMA-авторизацией + LoadUser (юзер должен существовать) ────
	userProtected := r.Group("/")
	userProtected.Use(auth)
	userProtected.Use(middleware.LoadUser(database))
	{
		userProtected.POST("/payment/create", handlers.CreatePayment(database))
		userProtected.POST("/profile/role", handlers.SetRole(database))

		// Клиент: создать заявку и посмотреть свои заказы
		userProtected.POST("/customer/orders", handlers.CustomerCreateOrder(database))
		userProtected.GET("/orders/my", handlers.GetCustomerOrders(database))

		// Исполнитель: принять заявку, посмотреть свои, обновить статус
		userProtected.POST("/executor/orders/:id/accept", handlers.AcceptOrder(database))
		userProtected.GET("/executor/orders/my", handlers.GetMyOrders(database))
		userProtected.PUT("/executor/orders/:id/status", handlers.ExecutorUpdateOrderStatus(database))
		userProtected.GET("/executor/achievements", handlers.GetAchievements(database))
	}

	// ── Админ-маршруты (tma auth + role=admin) ────────────────────────────────
	adminGroup := r.Group("/admin")
	adminGroup.Use(auth)
	adminGroup.Use(middleware.AdminOnly(database))
	{
		adminGroup.GET("/stats", handlers.AdminGetStats(database))

		adminGroup.GET("/catalog", handlers.AdminListCatalog(database))
		adminGroup.POST("/catalog", handlers.AdminCreateCatalogItem(database))
		adminGroup.PUT("/catalog/:id", handlers.AdminUpdateCatalogItem(database))
		adminGroup.DELETE("/catalog/:id", handlers.AdminDeleteCatalogItem(database))

		adminGroup.GET("/achievements", handlers.AdminListAchievements(database))
		adminGroup.POST("/achievements", handlers.AdminCreateAchievement(database))
		adminGroup.PUT("/achievements/:id", handlers.AdminUpdateAchievement(database))
		adminGroup.DELETE("/achievements/:id", handlers.AdminDeleteAchievement(database))

		adminGroup.GET("/users", handlers.AdminListUsers(database))
		adminGroup.PUT("/users/:id/role", handlers.AdminSetUserRole(database))
		adminGroup.POST("/users/:id/achievement", handlers.AdminGrantAchievement(database))

		adminGroup.GET("/orders", handlers.AdminListOrders(database))
		adminGroup.PUT("/orders/:id/status", handlers.AdminUpdateOrderStatus(database))

		// Заявки исполнителей из бота
		adminGroup.GET("/applications", handlers.AdminListApplications(database))
		adminGroup.POST("/applications/:id/approve", handlers.AdminApproveApplication(database))
		adminGroup.POST("/applications/:id/reject", handlers.AdminRejectApplication(database))
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
