package handlers

import (
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	initData, ok := middleware.CtxInitData(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации: initData не найдена"})
		return
	}

	tgUser := initData.User
	if tgUser.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка: данные пользователя Telegram не найдены"})
		return
	}

	database := db.DB
	var user models.User

	result := database.Where("telegram_id = ?", uint(tgUser.ID)).First(&user)
	if result.Error != nil {
		newUser := models.User{
			TelegramID: uint(tgUser.ID),
			FirstName:  tgUser.FirstName,
			LastName:   tgUser.LastName,
			Username:   tgUser.Username,
			IsPremium:  tgUser.IsPremium,
			PhotoURL:   tgUser.PhotoURL,
		}
		if err := db.DB.Create(&newUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать пользователя"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": newUser})
		return
	}

	// Возвращаем данные пользователя
	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
