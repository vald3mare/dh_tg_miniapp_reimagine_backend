package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateOrder — POST /orders, защищён X-API-Key (для Тильды и внешних источников)
func CreateOrder(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := os.Getenv("ORDERS_API_KEY")
		if apiKey != "" && c.GetHeader("X-API-Key") != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный API-ключ"})
			return
		}

		var body struct {
			ServiceType     string  `json:"service_type" form:"service_type"`
			Description     string  `json:"description" form:"description"`
			Price           float64 `json:"price" form:"price"`
			CustomerName    string  `json:"customer_name" form:"customer_name"`
			CustomerContact string  `json:"customer_contact" form:"customer_contact"`
			ScheduledAt     string  `json:"scheduled_at" form:"scheduled_at"`
		}

		if err := c.ShouldBind(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
			return
		}

		order := models.Order{
			ServiceType:     body.ServiceType,
			Description:     body.Description,
			Price:           body.Price,
			CustomerName:    body.CustomerName,
			CustomerContact: body.CustomerContact,
			Status:          "open",
		}

		if body.ScheduledAt != "" {
			t, err := time.Parse(time.RFC3339, body.ScheduledAt)
			if err == nil {
				order.ScheduledAt = &t
			}
		}

		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать заявку"})
			return
		}

		// Опциональное уведомление в Telegram
		notifyChatID := os.Getenv("EXECUTOR_NOTIFY_CHAT_ID")
		botToken := os.Getenv("BOT_TOKEN")
		if notifyChatID != "" && botToken != "" {
			go sendTelegramNotification(botToken, notifyChatID, order)
		}

		c.JSON(http.StatusOK, gin.H{"id": order.ID, "status": order.Status})
	}
}

// GetOpenOrders — GET /executor/orders, публичный список открытых заявок
func GetOpenOrders(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var orders []models.Order
		if err := db.Where("status = ?", "open").Order("created_at DESC").Find(&orders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить заявки"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"orders": orders, "total": len(orders)})
	}
}

// AcceptOrder — POST /executor/orders/:id/accept, защищённый
func AcceptOrder(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		initData, ok := middleware.CtxInitData(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		var executor models.User
		if err := db.Where("telegram_id = ?", uint(initData.User.ID)).First(&executor).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
			return
		}

		orderIDStr := c.Param("id")
		orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID заявки"})
			return
		}

		var order models.Order
		if err := db.First(&order, orderID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
			return
		}

		if order.Status != "open" {
			c.JSON(http.StatusConflict, gin.H{"error": "Заявка уже не в статусе open"})
			return
		}

		if err := db.Model(&order).Updates(map[string]any{
			"status":      "accepted",
			"executor_id": executor.ID,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить заявку"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"order": order})
	}
}

// GetMyOrders — GET /executor/orders/my, защищённый
func GetMyOrders(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		initData, ok := middleware.CtxInitData(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		var executor models.User
		if err := db.Where("telegram_id = ?", uint(initData.User.ID)).First(&executor).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
			return
		}

		var orders []models.Order
		if err := db.Where("executor_id = ?", executor.ID).Order("created_at DESC").Find(&orders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить заявки"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"orders": orders, "total": len(orders)})
	}
}

func sendTelegramNotification(botToken, chatID string, order models.Order) {
	text := "📋 Новая заявка!\n"
	text += "Тип: " + order.ServiceType + "\n"
	if order.CustomerName != "" {
		text += "Клиент: " + order.CustomerName + "\n"
	}
	if order.Description != "" {
		text += "Описание: " + order.Description + "\n"
	}
	if order.Price > 0 {
		text += "Цена: " + strconv.FormatFloat(order.Price, 'f', 0, 64) + "₽\n"
	}

	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WARN: не удалось сформировать уведомление: %v", err)
		return
	}

	resp, err := http.Post(
		"https://api.telegram.org/bot"+botToken+"/sendMessage",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Printf("WARN: не удалось отправить уведомление в Telegram: %v", err)
		return
	}
	defer resp.Body.Close()
}
