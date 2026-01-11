package main

import (
	"log"
	"os"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/handlers"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/yookassa"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Чтение токена бота из переменных окружения
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is not set")
	}

	// Инициализация базы данных
	if err := db.InitDB(); err != nil {
		log.Printf("WARNING: Failed to initialize database: %v - continuing without DB", err)
	} else {
		log.Println("Database initialized successfully")
	}

	// Инициализация ЮKassa клиента
	if err := yookassa.Init(); err != nil {
		log.Printf("WARNING: Failed to initialize Yookassa: %v", err)
	}

	log.Println("Server starting...")

	r := gin.New()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Логируем все запросы
	r.Use(gin.Logger())

	// Middleware авторизации — применяется ко всем защищённым роутам
	auth := middleware.AuthMiddleware(token)
	log.Println(auth)

	// Защищённые роуты (все под /)
	protected := r.Group("/")
	protected.Use(auth)
	{
		protected.POST("/", handlers.ShowInitData)
		protected.GET("/profile", handlers.GetProfile) // профиль
		protected.POST("/payment/create", handlers.CreatePayment)
		protected.GET("/payment/:payment_id", handlers.GetPaymentStatus)
		protected.POST("/payment/:payment_id/cancel", handlers.CancelPayment)
		protected.POST("/payment/:payment_id/capture", handlers.CapturePayment)
		protected.POST("/subscription/cancel", handlers.CancelSubscription)
	}

	// Вебхуки и платежные редиректы (открытые роуты)
	r.POST("/webhook/yookassa", handlers.YookassaWebhook)
	r.GET("/payment/success", handlers.PaymentSuccess)

	// Не защищённый health-check (для Timeweb и мониторинга)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // fallback для локального запуска
	}

	log.Printf("Listening on :%s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	if db.DB == nil {
		log.Fatal("DB is nil after InitDB")
	}
}
