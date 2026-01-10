package handlers

import (
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	initData, _ := middleware.CtxInitData(c.Request.Context())
	tgUser := initData.User

	var user models.User
	result := db.DB.Preload("Subscription").First(&user, "telegram_id = ?", tgUser.ID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
