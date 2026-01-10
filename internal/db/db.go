package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
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
		return fmt.Errorf("не заданы переменные окружения для БД")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("не удалось открыть соединение: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("ping к БД провалился: %w", err)
	}
	log.Println("PostgreSQL успешно подключена (pgx)")

	// 1. Создаём users
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
		return fmt.Errorf("ошибка создания users: %w", err)
	}
	log.Println("Таблица users создана/обновлена")

	// Ключевой момент: даём PostgreSQL время обновить каталог
	time.Sleep(100 * time.Millisecond) // 100ms пауза — обычно хватает
	db.Ping()                          // дополнительный пинг для сброса кэша

	// 2. Создаём subscriptions
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
		return fmt.Errorf("ошибка создания subscriptions: %w", err)
	}
	log.Println("Таблица subscriptions создана/обновлена")

	log.Println("PostgreSQL успешно подключена и мигрирована")
	return nil
}
