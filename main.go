package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/handlers"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// ===== Инициализация переменных окружения =====
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// ===== Инициализация базы данных =====
	database, err := db.InitDB()
	if err != nil {
		log.Printf("WARNING: Failed to initialize database: %v - continuing without DB", err)
	} else {
		log.Println("Database initialized successfully")
	}

	database.AutoMigrate(&models.User{}, &models.Subscription{})

	// if err := yookassa.Init(); err != nil {
	// 	log.Printf("WARNING: Failed to initialize Yookassa: %v", err)
	// }

	// Определяем роутер Gin + задаем свой формат логов
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Middleware авторизации — применяется ко всем защищённым роутам
	auth := middleware.AuthMiddleware(token)

	// Защищённые роуты (все под /)
	protected := r.Group("/")
	protected.Use(auth)
	{
		//protected.POST("/", handlers.ShowInitData)
		protected.GET("/", handlers.GetProfile)
		protected.GET("/profile", handlers.GetProfile)
		protected.POST("/payment/create", handlers.CreatePayment)
		protected.GET("/payment/:payment_id", handlers.GetPaymentStatus)
		protected.POST("/payment/:payment_id/cancel", handlers.CancelPayment)
		protected.POST("/payment/:payment_id/capture", handlers.CapturePayment)
		//protected.POST("/subscription/cancel", handlers.CancelSubscription)
	}

	// Вебхуки и платежные редиректы (открытые роуты)
	r.POST("/webhook/yookassa", handlers.YookassaWebhook)
	r.GET("/payment/success", handlers.PaymentSuccess)

	// Не защищённый health-check (для Timeweb и мониторинга)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("Listening on :%s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
