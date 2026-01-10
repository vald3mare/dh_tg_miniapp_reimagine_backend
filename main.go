package main

import (
	"log"
	"os"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/handlers"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is not set")
	}

	// Подключаем БД (если нужно — раскомментируй, когда будешь готов)
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
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

	// Защищённые роуты (все под /)
	protected := r.Group("/")
	protected.Use(auth)
	{
		protected.POST("/", handlers.ShowInitData)     // тестовый/авторизационный
		protected.GET("/profile", handlers.GetProfile) // профиль
		// protected.POST("/subscription/cancel", handlers.CancelSubscription)
		// protected.POST("/subscription/renew", handlers.RenewSubscription)
	}

	// Не защищённый health-check (для Timeweb и мониторинга)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Запуск сервера — самый последний шаг!
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // fallback для локального запуска
	}

	log.Printf("Listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	if db.DB == nil {
		log.Fatal("DB is nil after InitDB")
	}
}
