package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
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

	// Миграции через migrate
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("не удалось создать драйвер миграций: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../migrations", // путь к папке migrations относительно cmd/server
		"postgres", driver,
	)
	if err != nil {
		return fmt.Errorf("не удалось создать мигратор: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка миграции: %w", err)
	}

	log.Println("Миграции успешно применены (или уже актуальны)")

	DB = db
	return nil
}
