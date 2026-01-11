package db

import (
	"context"
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

// InitDB подключает к PostgreSQL и применяет миграции
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Логируем SQL-запросы
	})
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err = sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping к БД провалился: %w", err)
	}

	log.Println("PostgreSQL успешно подключена")

	// Миграции с контекстом и таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.WithContext(ctx).AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("ошибка миграции users: %w", err)
	}
	log.Println("Таблица users создана/обновлена")

	if err := db.WithContext(ctx).AutoMigrate(&models.Subscription{}); err != nil {
		return fmt.Errorf("ошибка миграции subscriptions: %w", err)
	}
	log.Println("Таблица subscriptions создана/обновлена")

	DB = db
	return nil
}

// GetDBWithContext возвращает db с контекстом (для использования в хендлерах)
func GetDBWithContext(ctx context.Context) *gorm.DB {
	return DB.WithContext(ctx)
}
