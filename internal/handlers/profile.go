package handlers

import (
	"database/sql"
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	initData, _ := middleware.CtxInitData(c.Request.Context())
	tgUser := initData.User

	query := `
		SELECT id, telegram_id, first_name, last_name, username, language_code, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	row := db.DB.QueryRow(query, tgUser.ID)

	var id int64
	var telegramID int64
	var firstName, lastName, username, languageCode sql.NullString
	var createdAt, updatedAt sql.NullTime

	err := row.Scan(&id, &telegramID, &firstName, &lastName, &username, &languageCode, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка БД"})
		}
		return
	}

	user := gin.H{
		"id":            id,
		"telegram_id":   telegramID,
		"first_name":    firstName.String,
		"last_name":     lastName.String,
		"username":      username.String,
		"language_code": languageCode.String,
		"created_at":    createdAt.Time,
		"updated_at":    updatedAt.Time,
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
