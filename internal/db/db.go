package db

import (
	"fmt"
	"log"
	"os"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	// Бери данные из ENV (Timeweb Cloud позволяет задавать их в панели)
	host := os.Getenv("DB_HOST") // например db.timeweb.cloud или localhost
	port := os.Getenv("DB_PORT") // 5432
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE") // disable или require

	if host == "" || user == "" || password == "" || dbname == "" {
		log.Fatal("Не заданы переменные окружения для БД (DB_HOST, DB_USER и т.д.)")
		return fmt.Errorf("database environment variables are not set")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		host, user, password, dbname, port, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Не удалось подключиться к PostgreSQL:", err)
		return err
	}

	// Автомиграция таблиц (создаёт/обновляет таблицы по моделям)
	// 1. Сначала создаём таблицу users (без FK)
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Printf("Ошибка миграции users: %v", err)
		return err
	}
	log.Println("Таблица users создана/обновлена")

	// 2. Затем subscriptions (с FK на users)
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		log.Printf("Ошибка миграции subscriptions: %v", err)
		return err
	}
	log.Println("Таблица subscriptions создана/обновлена")

	DB = db
	log.Println("PostgreSQL успешно подключена и мигрирована")
	return nil
}
