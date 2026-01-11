package handlers

import (
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/db"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"github.com/gin-gonic/gin"
)

// CancelSubscription отменяет подписку пользователя
func CancelSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	// Получаем данные инициализации из контекста
	initData, ok := middleware.CtxInitData(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Init data not found"})
		return
	}

	userID := initData.User.ID

	// Получаем БД контекст
	dbCtx := db.GetDBWithContext(ctx)

	// Находим подписку пользователя
	var subscription models.Subscription
	if err := dbCtx.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	// Отменяем подписку
	subscription.Active = false
	if err := dbCtx.Save(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription cancelled successfully",
		"subscription_id": subscription.ID,
		"status":          "cancelled",
	})
}
