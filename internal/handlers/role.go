package handlers

import (
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetRole — POST /profile/role, защищённый
func SetRole(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		initData, ok := middleware.CtxInitData(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		var body struct {
			Role string `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите role: customer или executor"})
			return
		}

		if body.Role != "customer" && body.Role != "executor" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимая роль"})
			return
		}

		var user models.User
		if err := db.Where("telegram_id = ?", uint(initData.User.ID)).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
			return
		}

		if err := db.Model(&user).Update("role", body.Role).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить роль"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"role": user.Role})
	}
}
