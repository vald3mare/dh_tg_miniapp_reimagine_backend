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
		// Был: 12 отдельных COUNT запросов.
		// Стало: 3 агрегатных запроса — каждый считает всё нужное за один проход.

		type userStats struct {
			Total     int64 `json:"total"`
			Customers int64 `json:"customers"`
			Executors int64 `json:"executors"`
		}
		var users userStats
		db.Raw(`
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE role = 'customer') AS customers,
				COUNT(*) FILTER (WHERE role = 'executor') AS executors
			FROM users WHERE deleted_at IS NULL
		`).Scan(&users)

		type orderStats struct {
			Total    int64 `json:"total"`
			Open     int64 `json:"open"`
			Accepted int64 `json:"accepted"`
			Done     int64 `json:"done"`
		}
		var orders orderStats
		db.Raw(`
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status = 'open')     AS open,
				COUNT(*) FILTER (WHERE status = 'accepted') AS accepted,
				COUNT(*) FILTER (WHERE status = 'done')     AS done
			FROM orders WHERE deleted_at IS NULL
		`).Scan(&orders)

		type paymentStats struct {
			Total         int64   `json:"total"`
			Succeeded     int64   `json:"succeeded"`
			TotalRevenue  float64 `json:"total_revenue"`
		}
		var payments paymentStats
		db.Raw(`
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status = 'succeeded') AS succeeded,
				COALESCE(SUM(amount) FILTER (WHERE status = 'succeeded'), 0) AS total_revenue
			FROM payments WHERE deleted_at IS NULL
		`).Scan(&payments)

		var catalogTotal int64
		db.Model(&models.CatalogItem{}).Count(&catalogTotal)

		type ServiceStat struct {
			Name  string `json:"name"`
			Count int64  `json:"count"`
		}
		var topServices []ServiceStat
		db.Raw(`
			SELECT ci.name, COUNT(p.id) AS count
			FROM payments p
			JOIN catalog_items ci ON ci.id = p.item_id
			WHERE p.status = 'succeeded' AND p.deleted_at IS NULL
			GROUP BY ci.name ORDER BY count DESC LIMIT 5
		`).Scan(&topServices)

		type PaymentRow struct {
			ID          uint    `json:"id"`
			Amount      float64 `json:"amount"`
			Currency    string  `json:"currency"`
			Status      string  `json:"status"`
			Description string  `json:"description"`
			CreatedAt   string  `json:"created_at"`
		}
		var recentPayments []PaymentRow
		db.Raw(`
			SELECT p.id, p.amount, p.currency, p.status, p.description,
			       TO_CHAR(p.created_at, 'DD.MM.YYYY HH24:MI') AS created_at
			FROM payments p
			WHERE p.deleted_at IS NULL
			ORDER BY p.created_at DESC
			LIMIT 20
		`).Scan(&recentPayments)

		c.JSON(http.StatusOK, gin.H{
			"users":           users,
			"orders":          orders,
			"payments":        payments,
			"catalog":         gin.H{"total": catalogTotal},
			"top_services":    topServices,
			"recent_payments": recentPayments,
		})
	}
}

// ── Каталог ───────────────────────────────────────────────────────────────────

func AdminListCatalog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := parsePagination(c)
		var total int64
		db.Model(&models.CatalogItem{}).Count(&total)
		var items []models.CatalogItem
		db.Limit(limit).Offset(offset).Find(&items)
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "has_more": int64(offset+limit) < total})
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
		InvalidateCatalogCache()
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
		InvalidateCatalogCache()
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
		InvalidateCatalogCache()
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
		limit, offset := parsePagination(c)
		var total int64
		db.Model(&models.User{}).Count(&total)
		var users []models.User
		db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&users)
		c.JSON(http.StatusOK, gin.H{"users": users, "total": total, "has_more": int64(offset+limit) < total})
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
		limit, offset := parsePagination(c)
		var total int64
		db.Model(&models.Order{}).Count(&total)
		var orders []models.Order
		db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&orders)
		c.JSON(http.StatusOK, gin.H{"orders": orders, "total": total, "has_more": int64(offset+limit) < total})
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
			Status string `json:"status" binding:"required,oneof=open accepted done canceled"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var order models.Order
		if err := db.First(&order, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
			return
		}

		if err := db.Model(&order).Update("status", body.Status).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус"})
			return
		}

		// При завершении заказа начисляем достижения исполнителю
		if body.Status == "done" && order.ExecutorID != nil {
			var executor models.User
			if err := db.First(&executor, *order.ExecutorID).Error; err == nil {
				executor.OrdersCompleted++
				if err := db.Model(&executor).Update("orders_completed", executor.OrdersCompleted).Error; err == nil {
					CheckAndGrantAchievements(db, &executor)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
