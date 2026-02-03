package main

// эта миграция создаёт таблицы User и Subscription

import (
	"log"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
)

func main() {
	database, err := db.InitDB()
	if err != nil {
		log.Printf("WARNING: Failed to initialize database: %v - continuing without DB", err)
	} else {
		log.Println("Database initialized successfully")
	}

	database.AutoMigrate(&models.User{})
	database.AutoMigrate(&models.Subscription{})
}
