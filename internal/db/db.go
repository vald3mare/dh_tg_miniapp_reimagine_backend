package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Глобальная переменная для доступа к БД
//var DB *gorm.DB

// InitDB подключает к PostgreSQL и применяет миграции
func InitDB() (*gorm.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.Println("WARNING: Не все переменные окружения для БД заданы — работаем без БД")
		return nil, fmt.Errorf("database environment variables not set")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		host, port, user, password, dbname, sslmode)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("WARNING: Ошибка подключения к PostgreSQL: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Printf("WARNING: Ошибка получения sql.DB: %v", err)
		return nil, err
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("PostgreSQL успешно подключена ")
	return database, nil
}
