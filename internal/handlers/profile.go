package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	initData, _ := middleware.CtxInitData(c.Request.Context())
	database, _ := db.InitDB()

	tgUser := initData.User
	var user models.User

	ctx := context.Background()
	result := database.Where("telegram_id = ?", tgUser.ID).First(&user)
	tgID, _ := gorm.G[models.User](database).Where("telegram_id = ?", tgUser.ID).Select("telegram_id").First(ctx)
	fmt.Println("TG ID:", tgID)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
