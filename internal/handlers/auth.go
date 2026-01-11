package handlers

import (
	"log"
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
)

func ShowInitData(c *gin.Context) {
	initData, ok := middleware.CtxInitData(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Init data not found"})
		return
	}

	tgUser := initData.User

	// По доке GORM: First с Where для поиска
	var user models.User
	if err := db.DB.Where("telegram_id = ?", tgUser.ID).First(&user).Error; err != nil {
		// Не найден — Create (по доке GORM: Create)
		user = models.User{
			TelegramID:   tgUser.ID,
			FirstName:    tgUser.FirstName,
			LastName:     tgUser.LastName,
			Username:     tgUser.Username,
			LanguageCode: tgUser.LanguageCode,
			IsPremium:    tgUser.IsPremium,
			PhotoURL:     tgUser.PhotoURL,
		}
		if err := db.DB.Create(&user).Error; err != nil {
			log.Printf("Ошибка создания пользователя: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
			return
		}
		log.Printf("Создан новый пользователь: ID=%d", user.ID)
	} else {
		// Найден — обновляем (по доке GORM: Save)
		user.FirstName = tgUser.FirstName
		user.LastName = tgUser.LastName
		user.Username = tgUser.Username
		user.LanguageCode = tgUser.LanguageCode
		user.IsPremium = tgUser.IsPremium
		user.PhotoURL = tgUser.PhotoURL
		if err := db.DB.Save(&user).Error; err != nil {
			log.Printf("Ошибка обновления пользователя: %v", err)
		} else {
			log.Printf("Обновлён пользователь: ID=%d", user.ID)
		}
	}

	// Загружаем подписку (по доке GORM: Preload)
	if err := db.DB.Preload("Subscription").First(&user).Error; err != nil {
		log.Printf("Ошибка Preload подписки: %v", err)
	}

	// Возвращаем
	c.JSON(http.StatusOK, gin.H{
		"user":      user,
		"auth_date": initData.AuthDate,
	})
}
