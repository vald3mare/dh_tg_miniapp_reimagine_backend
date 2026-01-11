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

	// Логируем, что читается (отладка ENV)
	log.Printf("DB_HOST: '%s'", host)
	log.Printf("DB_PORT: '%s'", port)
	log.Printf("DB_USER: '%s'", user)
	log.Printf("DB_NAME: '%s'", dbname)

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.Println("WARNING: Не все переменные окружения для БД заданы — работаем без БД")
		return nil // НЕ fatal — продолжаем без БД
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		host, port, user, password, dbname, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Printf("WARNING: Ошибка подключения к PostgreSQL: %v — работаем без БД", err)
		return nil // НЕ fatal
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("WARNING: Ошибка получения sql.DB: %v", err)
		return nil
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err = sqlDB.Ping(); err != nil {
		log.Printf("WARNING: Ping к БД провалился: %v — работаем без БД", err)
		return nil
	}

	log.Println("PostgreSQL успешно подключена")

	// Сначала users
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Printf("WARNING: Ошибка миграции users: %v", err)
	} else {
		log.Println("Таблица users мигрирована")
	}

	// Пауза для Postgres (чтобы каталог обновился)
	time.Sleep(500 * time.Millisecond)

	// Затем subscriptions
	if err := db.AutoMigrate(&models.Subscription{}); err != nil {
		log.Printf("WARNING: Ошибка миграции subscriptions: %v", err)
	} else {
		log.Println("Таблица subscriptions мигрирована")
	}

	DB = db
	log.Println("PostgreSQL успешно подключена и мигрирована")
	return nil
}

// GetDBWithContext — безопасный доступ к DB (не падает на nil)
func GetDBWithContext(ctx context.Context) *gorm.DB {
	if DB == nil {
		log.Printf("WARNING: DB is nil — запросы к БД будут проигнорированы")
		return &gorm.DB{} // dummy — чтобы не паниковать
	}
	return DB.WithContext(ctx)
}
