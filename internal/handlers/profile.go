package handlers

import (
	"log"
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// resolveRoles возвращает (primaryRole, rolesArray)
// Для админов — все 3 роли; для остальных синхронизирует Roles с Role.
func resolveRoles(telegramID uint, currentRole string, currentRoles []string) (string, []string) {
	if middleware.IsAdminTelegramID(telegramID) {
		return "admin", []string{"admin", "customer", "executor"}
	}
	if len(currentRoles) > 0 && containsStr(currentRoles, currentRole) {
		return currentRole, currentRoles
	}
	// fallback: одна роль
	role := currentRole
	if role == "" {
		role = "customer"
	}
	return role, []string{role}
}

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
			role, roles := resolveRoles(uint(tgUser.ID), "customer", nil)
			newUser := models.User{
				TelegramID: uint(tgUser.ID),
				FirstName:  tgUser.FirstName,
				LastName:   tgUser.LastName,
				Username:   tgUser.Username,
				IsPremium:  tgUser.IsPremium,
				PhotoURL:   tgUser.PhotoURL,
				Role:       role,
				Roles:      roles,
			}
			if err := db.Create(&newUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать пользователя"})
				return
			}
			log.Printf("Создан новый пользователь TelegramID=%d role=%s roles=%v", newUser.TelegramID, newUser.Role, newUser.Roles)
			c.JSON(http.StatusOK, gin.H{"user": newUser})
			return
		}

		// Обновляем все поля разом через Save — один UPDATE вместо трёх
		newRole, newRoles := resolveRoles(uint(tgUser.ID), user.Role, user.Roles)
		user.FirstName = tgUser.FirstName
		user.LastName = tgUser.LastName
		user.Username = tgUser.Username
		user.IsPremium = tgUser.IsPremium
		user.PhotoURL = tgUser.PhotoURL
		user.Role = newRole
		user.Roles = newRoles
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить пользователя"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}
