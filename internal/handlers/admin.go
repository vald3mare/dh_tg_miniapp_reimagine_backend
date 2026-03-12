package handlers

import (
	"net/http"
	"strconv"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ── Статистика ────────────────────────────────────────────────────────────────

func AdminGetStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var usersTotal, customersTotal, executorsTotal int64
		var ordersTotal, ordersOpen, ordersAccepted, ordersDone int64
		var paymentsTotal, paymentsSucceeded int64
		var catalogTotal int64

		db.Model(&models.User{}).Count(&usersTotal)
		db.Model(&models.User{}).Where("role = ?", "customer").Count(&customersTotal)
		db.Model(&models.User{}).Where("role = ?", "executor").Count(&executorsTotal)

		db.Model(&models.Order{}).Count(&ordersTotal)
		db.Model(&models.Order{}).Where("status = ?", "open").Count(&ordersOpen)
		db.Model(&models.Order{}).Where("status = ?", "accepted").Count(&ordersAccepted)
		db.Model(&models.Order{}).Where("status = ?", "done").Count(&ordersDone)

		db.Model(&models.Payment{}).Count(&paymentsTotal)
		db.Model(&models.Payment{}).Where("status = ?", "succeeded").Count(&paymentsSucceeded)

		db.Model(&models.CatalogItem{}).Count(&catalogTotal)

		// Топ услуг по платежам
		type ServiceStat struct {
			Name  string
			Count int64
		}
		var topServices []ServiceStat
		db.Raw(`
			SELECT ci.name, COUNT(p.id) as count
			FROM payments p
			JOIN catalog_items ci ON ci.id = p.item_id
			WHERE p.status = 'succeeded' AND p.deleted_at IS NULL
			GROUP BY ci.name ORDER BY count DESC LIMIT 5
		`).Scan(&topServices)

		c.JSON(http.StatusOK, gin.H{
			"users": gin.H{
				"total":     usersTotal,
				"customers": customersTotal,
				"executors": executorsTotal,
			},
			"orders": gin.H{
				"total":    ordersTotal,
				"open":     ordersOpen,
				"accepted": ordersAccepted,
				"done":     ordersDone,
			},
			"payments": gin.H{
				"total":     paymentsTotal,
				"succeeded": paymentsSucceeded,
			},
			"catalog": gin.H{
				"total": catalogTotal,
			},
			"top_services": topServices,
		})
	}
}

// ── Каталог ───────────────────────────────────────────────────────────────────

func AdminListCatalog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.CatalogItem
		db.Find(&items)
		c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
	}
}

func AdminCreateCatalogItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name            string  `json:"name" binding:"required"`
			Description     string  `json:"description"`
			FullDescription string  `json:"full_description"`
			Price           float64 `json:"price"`
			ImageURL        string  `json:"image_url"`
			Type            string  `json:"type"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		item := models.CatalogItem{
			Name:            body.Name,
			Description:     body.Description,
			FullDescription: body.FullDescription,
			Price:           body.Price,
			ImageURL:        body.ImageURL,
			Type:            body.Type,
		}
		if err := db.Create(&item).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать услугу"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"item": item})
	}
}

func AdminUpdateCatalogItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		var item models.CatalogItem
		if err := db.First(&item, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Услуга не найдена"})
			return
		}
		var body struct {
			Name            *string  `json:"name"`
			Description     *string  `json:"description"`
			FullDescription *string  `json:"full_description"`
			Price           *float64 `json:"price"`
			ImageURL        *string  `json:"image_url"`
			Type            *string  `json:"type"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates := map[string]any{}
		if body.Name != nil            { updates["name"] = *body.Name }
		if body.Description != nil     { updates["description"] = *body.Description }
		if body.FullDescription != nil { updates["full_description"] = *body.FullDescription }
		if body.Price != nil           { updates["price"] = *body.Price }
		if body.ImageURL != nil        { updates["image_url"] = *body.ImageURL }
		if body.Type != nil            { updates["type"] = *body.Type }
		db.Model(&item).Updates(updates)
		c.JSON(http.StatusOK, gin.H{"item": item})
	}
}

func AdminDeleteCatalogItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		if err := db.Delete(&models.CatalogItem{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось удалить"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ── Ачивки ────────────────────────────────────────────────────────────────────

func AdminListAchievements(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.Achievement
		db.Find(&items)
		c.JSON(http.StatusOK, gin.H{"achievements": items})
	}
}

func AdminCreateAchievement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Key           string `json:"key" binding:"required"`
			Name          string `json:"name" binding:"required"`
			Description   string `json:"description"`
			IconEmoji     string `json:"icon_emoji"`
			ConditionType string `json:"condition_type"`
			Threshold     int    `json:"threshold"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		item := models.Achievement{
			Key:           body.Key,
			Name:          body.Name,
			Description:   body.Description,
			IconEmoji:     body.IconEmoji,
			ConditionType: body.ConditionType,
			Threshold:     body.Threshold,
		}
		if err := db.Create(&item).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать ачивку"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"achievement": item})
	}
}

func AdminUpdateAchievement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		var item models.Achievement
		if err := db.First(&item, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ачивка не найдена"})
			return
		}
		var body struct {
			Name          *string `json:"name"`
			Description   *string `json:"description"`
			IconEmoji     *string `json:"icon_emoji"`
			ConditionType *string `json:"condition_type"`
			Threshold     *int    `json:"threshold"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates := map[string]any{}
		if body.Name != nil          { updates["name"] = *body.Name }
		if body.Description != nil   { updates["description"] = *body.Description }
		if body.IconEmoji != nil     { updates["icon_emoji"] = *body.IconEmoji }
		if body.ConditionType != nil { updates["condition_type"] = *body.ConditionType }
		if body.Threshold != nil     { updates["threshold"] = *body.Threshold }
		db.Model(&item).Updates(updates)
		c.JSON(http.StatusOK, gin.H{"achievement": item})
	}
}

func AdminDeleteAchievement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		if err := db.Delete(&models.Achievement{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось удалить"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ── Пользователи ──────────────────────────────────────────────────────────────

func AdminListUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		db.Order("created_at DESC").Find(&users)
		c.JSON(http.StatusOK, gin.H{"users": users, "total": len(users)})
	}
}

func AdminSetUserRole(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		var body struct {
			Role string `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Role != "customer" && body.Role != "executor" && body.Role != "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимая роль"})
			return
		}
		if err := db.Model(&models.User{}).Where("id = ?", id).Update("role", body.Role).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить роль"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// AdminGrantAchievement — выдать ачивку пользователю вручную
func AdminGrantAchievement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
			return
		}
		var body struct {
			AchievementID uint `json:"achievement_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Проверяем что ачивки ещё нет
		var count int64
		db.Model(&models.UserAchievement{}).
			Where("user_id = ? AND achievement_id = ?", userID, body.AchievementID).
			Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Ачивка уже выдана"})
			return
		}

		ua := models.UserAchievement{
			UserID:        uint(userID),
			AchievementID: body.AchievementID,
		}
		if err := db.Create(&ua).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось выдать ачивку"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ── Заявки ────────────────────────────────────────────────────────────────────

func AdminListOrders(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var orders []models.Order
		db.Order("created_at DESC").Find(&orders)
		c.JSON(http.StatusOK, gin.H{"orders": orders, "total": len(orders)})
	}
}

func AdminUpdateOrderStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}
		var body struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		allowed := map[string]bool{"open": true, "accepted": true, "done": true, "canceled": true}
		if !allowed[body.Status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимый статус"})
			return
		}
		if err := db.Model(&models.Order{}).Where("id = ?", id).Update("status", body.Status).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
