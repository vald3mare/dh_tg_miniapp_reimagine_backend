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
	"gorm.io/gorm"
)

// В main.go, после db.AutoMigrate(&models.CatalogItem{})
func seedCatalog(db *gorm.DB) {
	var count int64
	db.Model(&models.CatalogItem{}).Count(&count)
	if count == 0 {
		items := []models.CatalogItem{
			{Name: "Выгул собак", Description: "Полная забота о вашем питомце во время прогулки: от экипировки до игр и обучения", FullDescription: "Выгульщик приходит домой сам — с экипировкой или использует вашу.\nОдевает собаку, берёт вкусняшки, пакеты, поводки.\nУчитывает возраст, темперамент, повадки, особенности прогулок.", Price: 890, ImageURL: "https://s3.twcstorage.ru/dh-s3-storage/shutterstock_1531627.png", Type: "service"},
			{Name: "Выгул животных", Description: "Часовая прогулка", FullDescription: "Часовая прогулка с вашим питомцем.\nВключает: кормление, игры.\nЦена за час.", Price: 500, ImageURL: "https://example.com/dog-walk.jpg", Type: "service"},
			{Name: "Корм для собак", Description: "Премиум корм 10кг", FullDescription: "Премиум корм 10кг.\nСостав: мясо, витамины.\nРекомендации: для взрослых собак.", Price: 3000, ImageURL: "https://example.com/dog-food.jpg", Type: "product"},
		}
		db.Create(&items)
		fmt.Println("Seed данные добавлены в каталог")
	}
}

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

	database.AutoMigrate(&models.User{}, &models.Subscription{}, &models.CatalogItem{})
	seedCatalog(database)

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
		protected.GET("/", handlers.GetProfile)
		protected.GET("/profile", handlers.GetProfile)
	}

	r.POST("/payment/create", handlers.CreateTestPayment(database))
	r.GET("/catalog", handlers.GetCatalog(database))

	// Не защищённый health-check (для Timeweb и мониторинга)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("Listening on :%s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
