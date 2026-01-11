package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
)

func ShowInitData(c *gin.Context) {
	ctx := c.Request.Context() // Контекст из Gin (отменяется при закрытии запроса)

	// Добавляем таймаут на весь запрос (5 секунд — по доке GORM)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	initData, ok := middleware.CtxInitData(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Init data not found"})
		return
	}

	tgUser := initData.User

	// Используем GetDBWithContext для всех операций
	dbCtx := db.GetDBWithContext(ctx)

	var user models.User
	if err := dbCtx.Where("telegram_id = ?", tgUser.ID).First(&user).Error; err != nil {
		// Не найден — создаём нового
		user = models.User{
			TelegramID:   tgUser.ID,
			FirstName:    tgUser.FirstName,
			LastName:     tgUser.LastName,
			Username:     tgUser.Username,
			LanguageCode: tgUser.LanguageCode,
			IsPremium:    tgUser.IsPremium,
			PhotoURL:     tgUser.PhotoURL,
		}
		if err := dbCtx.Create(&user).Error; err != nil {
			log.Printf("Ошибка создания пользователя: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
			return
		}
		log.Printf("Создан новый пользователь: ID=%d", user.ID)
	} else {
		// Обновляем
		user.FirstName = tgUser.FirstName
		user.LastName = tgUser.LastName
		user.Username = tgUser.Username
		user.LanguageCode = tgUser.LanguageCode
		user.IsPremium = tgUser.IsPremium
		user.PhotoURL = tgUser.PhotoURL
		if err := dbCtx.Save(&user).Error; err != nil {
			log.Printf("Ошибка обновления пользователя: %v", err)
		} else {
			log.Printf("Обновлён пользователь: ID=%d", user.ID)
		}
	}

	// Preload подписки с контекстом
	if err := dbCtx.Preload("Subscription").First(&user).Error; err != nil {
		log.Printf("Ошибка Preload подписки: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":      user,
		"auth_date": initData.AuthDate,
	})
}
