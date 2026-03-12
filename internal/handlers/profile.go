package handlers

import (
	"log"
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		initData, ok := middleware.CtxInitData(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации: initData не найдена"})
			return
		}

		tgUser := initData.User
		if tgUser.ID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Данные пользователя Telegram не найдены"})
			return
		}

		var user models.User
		result := db.Where("telegram_id = ?", uint(tgUser.ID)).First(&user)

		if result.Error != nil {
			// Пользователь не найден — создаём
			newUser := models.User{
				TelegramID: uint(tgUser.ID),
				FirstName:  tgUser.FirstName,
				LastName:   tgUser.LastName,
				Username:   tgUser.Username,
				IsPremium:  tgUser.IsPremium,
				PhotoURL:   tgUser.PhotoURL,
			}
			if err := db.Create(&newUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать пользователя"})
				return
			}
			log.Printf("Создан новый пользователь TelegramID=%d", newUser.TelegramID)
			c.JSON(http.StatusOK, gin.H{"user": newUser})
			return
		}

		// Пользователь найден — обновляем только изменяемые поля профиля Telegram.
		// Используем Updates (не Save) чтобы не затирать поля вроде подписки.
		updates := map[string]any{
			"first_name": tgUser.FirstName,
			"last_name":  tgUser.LastName,
			"username":   tgUser.Username,
			"is_premium": tgUser.IsPremium,
			"photo_url":  tgUser.PhotoURL,
		}
		if err := db.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить пользователя"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}
