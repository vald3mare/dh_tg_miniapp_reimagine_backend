package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
)

// IsAdminTelegramID проверяет, входит ли telegramID в список ADMIN_TELEGRAM_IDS из env
func IsAdminTelegramID(telegramID uint) bool {
	raw := os.Getenv("ADMIN_TELEGRAM_IDS")
	if raw == "" {
		return false
	}
	for _, part := range strings.Split(raw, ",") {
		id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64)
		if err == nil && uint(id) == telegramID {
			return true
		}
	}
	return false
}

// AdminOnly — middleware, пропускает только пользователей с role=admin в БД
func AdminOnly(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		initData, ok := CtxInitData(c.Request.Context())
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		var user models.User
		if err := db.Where("telegram_id = ?", uint(initData.User.ID)).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Пользователь не найден"})
			return
		}

		if user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Доступ запрещён"})
			return
		}

		c.Next()
	}
}
