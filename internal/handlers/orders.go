package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateOrder — POST /orders, защищён X-API-Key (для Тильды и внешних источников).
// Ключ берётся из ORDERS_API_KEY. Если переменная не задана — запрос не проходит,
// если только не выставлен ORDERS_AUTH_DISABLED=true.
func CreateOrder(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := os.Getenv("ORDERS_API_KEY")
		authDisabled := os.Getenv("ORDERS_AUTH_DISABLED") == "true"

		if !authDisabled {
			if apiKey == "" || c.GetHeader("X-API-Key") != apiKey {
				c.JSON(http.StatusForbidden, gin.H{"error": "Неверный или отсутствующий API-ключ"})
				return
			}
		}

		var body struct {
			ServiceType     string  `json:"service_type" binding:"required" form:"service_type"`
			Description     string  `json:"description"                     form:"description"`
			Price           float64 `json:"price"        binding:"min=0"    form:"price"`
			CustomerName    string  `json:"customer_name"                   form:"customer_name"`
			CustomerContact string  `json:"customer_contact"                form:"customer_contact"`
			ScheduledAt     string  `json:"scheduled_at"                    form:"scheduled_at"`
		}

		if err := c.ShouldBind(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		limit, offset := parsePagination(c)

		var total int64
		db.Model(&models.Order{}).Where("status = ?", "open").Count(&total)

		var orders []models.Order
		if err := db.Where("status = ?", "open").
			Order("created_at DESC").
			Limit(limit).Offset(offset).
			Find(&orders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить заявки"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"orders":   orders,
			"total":    total,
			"has_more": int64(offset+limit) < total,
		})
	}
}

// AcceptOrder — POST /executor/orders/:id/accept, защищённый.
// Использует SELECT FOR UPDATE в транзакции, чтобы исключить race condition.
func AcceptOrder(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		executor, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		orderIDStr := c.Param("id")
		orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID заявки"})
			return
		}

		var acceptedOrder models.Order
		txErr := db.Transaction(func(tx *gorm.DB) error {
			var order models.Order
			// SELECT ... FOR UPDATE — блокируем строку на время транзакции
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&order, orderID).Error; err != nil {
				return err
			}
			if order.Status != "open" {
				return errors.New("заявка уже не в статусе open")
			}
			order.Status = "accepted"
			order.ExecutorID = &executor.ID
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			acceptedOrder = order
			return nil
		})

		if txErr != nil {
			if errors.Is(txErr, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
				return
			}
			c.JSON(http.StatusConflict, gin.H{"error": txErr.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"order": acceptedOrder})
	}
}

// GetMyOrders — GET /executor/orders/my, защищённый
func GetMyOrders(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		executor, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		limit, offset := parsePagination(c)

		var total int64
		db.Model(&models.Order{}).Where("executor_id = ?", executor.ID).Count(&total)

		var orders []models.Order
		if err := db.Where("executor_id = ?", executor.ID).
			Order("created_at DESC").
			Limit(limit).Offset(offset).
			Find(&orders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить заявки"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"orders":   orders,
			"total":    total,
			"has_more": int64(offset+limit) < total,
		})
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
