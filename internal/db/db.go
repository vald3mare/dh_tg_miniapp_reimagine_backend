package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx как драйвер
)

var DB *sql.DB

func InitDB() error {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	if host == "" || user == "" || password == "" || dbname == "" {
		return fmt.Errorf("не заданы переменные окружения для БД (DB_HOST, DB_USER и т.д.)")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("не удалось открыть соединение: %w", err)
	}

	// Проверка подключения
	if err = db.Ping(); err != nil {
		return fmt.Errorf("ping к БД провалился: %w", err)
	}

	log.Println("PostgreSQL успешно подключена (pgx)")

	// 1. Создаём таблицу users (без зависимостей)
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id BIGSERIAL PRIMARY KEY,
            telegram_id BIGINT UNIQUE NOT NULL,
            first_name VARCHAR(255),
            last_name VARCHAR(255),
            username VARCHAR(255),
            language_code VARCHAR(10),
            is_premium BOOLEAN,
            photo_url VARCHAR(512),
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы users: %w", err)
	}
	log.Println("Таблица users создана/обновлена")

	// 2. Создаём таблицу subscriptions (с FK на users)
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS subscriptions (
            id BIGSERIAL PRIMARY KEY,
            user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            plan VARCHAR(50) DEFAULT 'free',
            active BOOLEAN DEFAULT false,
            start_date TIMESTAMPTZ,
            end_date TIMESTAMPTZ,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы subscriptions: %w", err)
	}
	log.Println("Таблица subscriptions создана/обновлена")

	log.Println("PostgreSQL успешно подключена и мигрирована")
	return nil
}
