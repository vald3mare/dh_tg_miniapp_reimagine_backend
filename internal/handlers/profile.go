package handlers

import (
	"log"
	"net/http"
	"strconv"

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

// UpdateProfileSettings — PATCH /profile/settings
// Обновляет display_name, city, avatar_data_url — данные хранятся в БД за юзером.
func UpdateProfileSettings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			return
		}

		var body struct {
			DisplayName   *string `json:"display_name"`
			City          *string `json:"city"`
			AvatarDataURL *string `json:"avatar_data_url"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := map[string]any{}
		if body.DisplayName != nil   { updates["display_name"]    = *body.DisplayName }
		if body.City != nil          { updates["city"]             = *body.City }
		if body.AvatarDataURL != nil { updates["avatar_data_url"] = *body.AvatarDataURL }

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Нечего обновлять"})
			return
		}

		if err := db.Model(user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить профиль"})
			return
		}

		// Обновляем локальную копию для ответа
		if body.DisplayName != nil   { user.DisplayName   = *body.DisplayName }
		if body.City != nil          { user.City          = *body.City }
		if body.AvatarDataURL != nil { user.AvatarDataURL = *body.AvatarDataURL }

		c.JSON(http.StatusOK, gin.H{
			"display_name":    user.DisplayName,
			"city":            user.City,
			"avatar_data_url": user.AvatarDataURL,
		})
	}
}

// GetPets — GET /profile/pets
func GetPets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			return
		}
		var pets []models.Pet
		db.Where("user_id = ?", user.ID).Order("created_at ASC").Find(&pets)
		c.JSON(http.StatusOK, gin.H{"pets": pets})
	}
}

// AddPet — POST /profile/pets
func AddPet(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			return
		}

		var body struct {
			Name  string `json:"name"  binding:"required"`
			Emoji string `json:"emoji"`
			Breed string `json:"breed"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		emoji := body.Emoji
		if emoji == "" { emoji = "🐾" }

		pet := models.Pet{UserID: user.ID, Name: body.Name, Emoji: emoji, Breed: body.Breed}
		if err := db.Create(&pet).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось добавить питомца"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"pet": pet})
	}
}

// DeletePet — DELETE /profile/pets/:id
func DeletePet(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			return
		}

		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		result := db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Pet{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления"})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Питомец не найден"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
