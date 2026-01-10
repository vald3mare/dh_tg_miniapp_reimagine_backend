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

	// Ищем пользователя по Telegram ID
	var user models.User
	result := db.DB.Where("telegram_id = ?", tgUser.ID).First(&user)

	if result.Error != nil {
		// Пользователь не найден — создаём нового
		user = models.User{
			TelegramID:   tgUser.ID,
			FirstName:    tgUser.FirstName,
			LastName:     tgUser.LastName,
			Username:     tgUser.Username,
			LanguageCode: tgUser.LanguageCode,
			IsPremium:    tgUser.IsPremium,
			PhotoURL:     tgUser.PhotoURL,
		}
		db.DB.Create(&user)
		log.Printf("Создан новый пользователь: ID=%d", user.ID)
	} else {
		// Обновляем существующие данные из Telegram
		user.FirstName = tgUser.FirstName
		user.LastName = tgUser.LastName
		user.Username = tgUser.Username
		user.LanguageCode = tgUser.LanguageCode
		user.IsPremium = tgUser.IsPremium
		user.PhotoURL = tgUser.PhotoURL
		db.DB.Save(&user)
		log.Printf("Обновлён пользователь: ID=%d", user.ID)
	}

	// Возвращаем полный профиль (с подпиской, если есть)
	db.DB.Preload("Subscription").First(&user, user.ID)

	c.JSON(http.StatusOK, gin.H{
		"user":      user,
		"auth_date": initData.AuthDate,
	})
}
