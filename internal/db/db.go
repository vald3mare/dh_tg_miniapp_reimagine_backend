package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() error {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return fmt.Errorf("не заданы переменные окружения для БД")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		host, port, user, password, dbname, sslmode)

	// Открываем соединение (по доке GORM: Config с Logger для отладки)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Логируем SQL-запросы
	})
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	// Получаем sql.DB для настройки пула (по доке GORM: performance)
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения sql.DB: %w", err)
	}

	// Настройка пула (по доке GORM: max open, idle, lifetime)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Проверка подключения (по доке: Ping)
	if err = sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping к БД провалился: %w", err)
	}

	log.Println("PostgreSQL успешно подключена")

	// Миграции (по доке GORM: AutoMigrate с порядком)
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("ошибка миграции users: %w", err)
	}
	log.Println("Таблица users мигрирована")

	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		return fmt.Errorf("ошибка миграции subscriptions: %w", err)
	}
	log.Println("Таблица subscriptions мигрирована")

	DB = db
	return nil
}
