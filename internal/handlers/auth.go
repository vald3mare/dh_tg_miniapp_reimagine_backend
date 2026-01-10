package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"

	"github.com/gin-gonic/gin"
)

func ShowInitData(c *gin.Context) {
	initData, ok := middleware.CtxInitData(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Init data not found"})
		return
	}

	tgUser := initData.User

	// Ищем пользователя по Telegram ID
	query := `
		SELECT id, telegram_id, first_name, last_name, username, language_code, is_premium, photo_url, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	row := db.DB.QueryRow(query, tgUser.ID)

	var id int64
	var telegramID int64
	var firstName, lastName, username, languageCode, photoURL sql.NullString
	var isPremium sql.NullBool
	var createdAt, updatedAt time.Time

	err := row.Scan(&id, &telegramID, &firstName, &lastName, &username, &languageCode, &isPremium, &photoURL, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		// Пользователь не найден — создаём нового
		insertQuery := `
			INSERT INTO users (telegram_id, first_name, last_name, username, language_code, is_premium, photo_url, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`

		row := db.DB.QueryRow(insertQuery, tgUser.ID, tgUser.FirstName, tgUser.LastName, tgUser.Username, tgUser.LanguageCode, tgUser.IsPremium, tgUser.PhotoURL)
		err := row.Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать пользователя"})
			return
		}
		log.Printf("Создан новый пользователь: ID=%d", id)

	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка БД"})
		return
	} else {
		// Обновляем существующие данные из Telegram
		updateQuery := `
			UPDATE users
			SET first_name = $1, last_name = $2, username = $3, language_code = $4, is_premium = $5, photo_url = $6, updated_at = NOW()
			WHERE id = $7
		`
		_, err := db.DB.Exec(updateQuery, tgUser.FirstName, tgUser.LastName, tgUser.Username, tgUser.LanguageCode, tgUser.IsPremium, tgUser.PhotoURL, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить пользователя"})
			return
		}
		log.Printf("Обновлён пользователь: ID=%d", id)
	}

	// Формируем ответ
	user := gin.H{
		"id":            id,
		"telegram_id":   telegramID,
		"first_name":    firstName.String,
		"last_name":     lastName.String,
		"username":      username.String,
		"language_code": languageCode.String,
		"is_premium":    isPremium.Bool,
		"photo_url":     photoURL.String,
		"created_at":    createdAt,
		"updated_at":    updatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"user":      user,
		"auth_date": initData.AuthDate,
	})
}
