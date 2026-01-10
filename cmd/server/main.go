package main

import (
	//"context"
	"log"
	"os"
	//"strings"
	//"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	//initdata "github.com/telegram-mini-apps/init-data-golang"

	//"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"          // ← подключение БД (добавь позже)
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/handlers"    // ← твои хендлеры
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable is not set")
	}

	log.Println("Server starting...")

	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(gin.Logger())
	r.Use(middleware.AuthMiddleware(token))
	//r.POST("/", handlers.ShowInitData)

	protected := r.Group("/")
	{
		protected.POST("/", handlers.ShowInitData) // текущий тестовый
		protected.GET("/profile", handlers.GetProfile) // новый для профиля
		// protected.POST("/subscription/cancel", handlers.CancelSubscription)
		// protected.POST("/subscription/renew", handlers.RenewSubscription)
	}

	// Не защищённый health-check (для Timeweb)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	if err := r.Run(":3000"); err != nil {
		panic(err)
	}
}
