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
	ctx := c.Request.Context()

	initData, ok := middleware.CtxInitData(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Init data not found"})
		return
	}

	tgUser := initData.User

	// Проверяем, доступна ли БД
	if db.DB == nil {
		log.Println("WARNING: БД недоступна — возвращаем данные только из Telegram")
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":         tgUser.ID,
				"first_name": tgUser.FirstName,
				"last_name":  tgUser.LastName,
				"username":   tgUser.Username,
				"language":   tgUser.LanguageCode,
				"is_premium": tgUser.IsPremium,
				"photo_url":  tgUser.PhotoURL,
			},
			"auth_date": initData.AuthDate,
			"note":      "DB not available",
		})
		return
	}

	dbCtx := db.GetDBWithContext(ctx)

	var user models.User
	if err := dbCtx.Where("telegram_id = ?", tgUser.ID).First(&user).Error; err != nil {
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
			c.JSON(http.StatusInternalServerError, gin.H{"message": "DB write error"})
			return
		}
		log.Printf("Создан новый пользователь: ID=%d", user.ID)
	} else {
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

	if err := dbCtx.Preload("Subscription").First(&user).Error; err != nil {
		log.Printf("Ошибка Preload подписки: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":      user,
		"auth_date": initData.AuthDate,
	})
}
