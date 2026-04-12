package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateOrder — POST /orders, защищён X-API-Key (для Тильды и внешних источников).
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
			if t, err := time.Parse(time.RFC3339, body.ScheduledAt); err == nil {
				order.ScheduledAt = &t
			}
		}

		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать заявку"})
			return
		}

		notifyExecutorChat(order)
		c.JSON(http.StatusOK, gin.H{"id": order.ID, "status": order.Status})
	}
}

// CustomerCreateOrder — POST /customer/orders, защищённый (TMA auth + LoadUser).
// Клиент создаёт заявку из мини-аппа; CustomerID и имя берутся из профиля Telegram.
func CustomerCreateOrder(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		customer, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		var body struct {
			ServiceType string  `json:"service_type" binding:"required"`
			Description string  `json:"description"`
			Price       float64 `json:"price"        binding:"min=0"`
			ScheduledAt string  `json:"scheduled_at"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		name := strings.TrimSpace(customer.FirstName + " " + customer.LastName)
		if name == "" {
			name = customer.Username
		}
		contact := ""
		if customer.Username != "" {
			contact = "@" + customer.Username
		}

		order := models.Order{
			CustomerID:      &customer.ID,
			ServiceType:     body.ServiceType,
			Description:     body.Description,
			Price:           body.Price,
			CustomerName:    name,
			CustomerContact: contact,
			Status:          "open",
		}

		if body.ScheduledAt != "" {
			if t, err := time.Parse(time.RFC3339, body.ScheduledAt); err == nil {
				order.ScheduledAt = &t
			}
		}

		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать заявку"})
			return
		}

		// Уведомляем клиента
		go NotifyUser(customer.TelegramID, fmt.Sprintf(
			"✅ <b>Заявка принята!</b>\n\n"+
				"Услуга: <b>%s</b>\n"+
				"Статус: ожидает исполнителя\n\n"+
				"Мы уведомим вас, как только исполнитель возьмёт заявку в работу. 🐾",
			order.ServiceType,
		))

		// Уведомляем чат исполнителей
		notifyExecutorChat(order)

		c.JSON(http.StatusOK, gin.H{"id": order.ID, "status": order.Status})
	}
}

// GetCustomerOrders — GET /orders/my, защищённый.
// Возвращает заказы текущего клиента (customer_id = текущий пользователь).
func GetCustomerOrders(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		customer, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		limit, offset := parsePagination(c)

		var total int64
		db.Model(&models.Order{}).Where("customer_id = ?", customer.ID).Count(&total)

		var orders []models.Order
		if err := db.Where("customer_id = ?", customer.ID).
			Order("created_at DESC").
			Limit(limit).Offset(offset).
			Find(&orders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить заказы"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"orders":   orders,
			"total":    total,
			"has_more": int64(offset+limit) < total,
		})
	}
}

// GetOpenOrders — GET /executor/orders, публичный список открытых заявок.
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
// SELECT FOR UPDATE исключает race condition при одновременном принятии заявки.
func AcceptOrder(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		executor, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID заявки"})
			return
		}

		var acceptedOrder models.Order
		txErr := db.Transaction(func(tx *gorm.DB) error {
			var order models.Order
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

		// Уведомляем клиента о том, кто взял заявку
		if acceptedOrder.CustomerID != nil {
			var customer models.User
			if db.First(&customer, *acceptedOrder.CustomerID).Error == nil {
				executorName := strings.TrimSpace(executor.FirstName + " " + executor.LastName)
				if executorName == "" {
					executorName = executor.Username
				}
				go NotifyUser(customer.TelegramID, fmt.Sprintf(
					"🐕 <b>Исполнитель найден!</b>\n\n"+
						"Ваша заявка «<b>%s</b>» принята исполнителем <b>%s</b>.\n"+
						"Ожидайте связи с исполнителем. 🐾",
					acceptedOrder.ServiceType, executorName,
				))
			}
		}

		c.JSON(http.StatusOK, gin.H{"order": acceptedOrder})
	}
}

// GetMyOrders — GET /executor/orders/my, защищённый.
// Возвращает заявки, которые взял текущий исполнитель.
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

// ExecutorUpdateOrderStatus — PUT /executor/orders/:id/status, защищённый.
// Исполнитель обновляет статус своей заявки.
// Допустимые переходы: accepted → in_progress → done
func ExecutorUpdateOrderStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		executor, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID заявки"})
			return
		}

		var body struct {
			Status string `json:"status" binding:"required,oneof=in_progress done"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status должен быть in_progress или done"})
			return
		}

		var order models.Order
		if err := db.First(&order, orderID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
			return
		}

		// Только исполнитель, назначенный на заявку, может её обновлять
		if order.ExecutorID == nil || *order.ExecutorID != executor.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Эта заявка не назначена на вас"})
			return
		}

		if order.Status == "done" || order.Status == "canceled" {
			c.JSON(http.StatusConflict, gin.H{"error": "Статус заявки уже финальный"})
			return
		}

		if err := db.Model(&order).Update("status", body.Status).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус"})
			return
		}

		// При завершении заказа: начисляем ачивки и уведомляем клиента
		if body.Status == "done" {
			executor.OrdersCompleted++
			db.Model(executor).Update("orders_completed", executor.OrdersCompleted)
			CheckAndGrantAchievements(db, executor)

			if order.CustomerID != nil {
				var customer models.User
				if db.First(&customer, *order.CustomerID).Error == nil {
					go NotifyUser(customer.TelegramID, fmt.Sprintf(
						"✅ <b>Заявка выполнена!</b>\n\n"+
							"Услуга «<b>%s</b>» завершена.\n"+
							"Спасибо, что пользуетесь Собачьим Счастьем! 🐾",
						order.ServiceType,
					))
				}
			}
		}

		if body.Status == "in_progress" {
			if order.CustomerID != nil {
				var customer models.User
				if db.First(&customer, *order.CustomerID).Error == nil {
					go NotifyUser(customer.TelegramID, fmt.Sprintf(
						"🐕 <b>Исполнитель приступил к работе!</b>\n\n"+
							"Ваша заявка «<b>%s</b>» выполняется прямо сейчас. 🏃",
						order.ServiceType,
					))
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "status": body.Status})
	}
}

// notifyExecutorChat — отправляет уведомление в групповой чат исполнителей
// (EXECUTOR_NOTIFY_CHAT_ID) при появлении новой заявки.
func notifyExecutorChat(order models.Order) {
	notifyChatID := os.Getenv("EXECUTOR_NOTIFY_CHAT_ID")
	botToken := os.Getenv("BOT_TOKEN")
	if notifyChatID == "" || botToken == "" {
		return
	}

	text := "📋 <b>Новая заявка!</b>\n"
	text += "Тип: " + order.ServiceType + "\n"
	if order.CustomerName != "" {
		text += "Клиент: " + order.CustomerName + "\n"
	}
	if order.Description != "" {
		text += "Описание: " + order.Description + "\n"
	}
	if order.Price > 0 {
		text += "Цена: " + strconv.FormatFloat(order.Price, 'f', 0, 64) + " ₽\n"
	}

	chatIDInt, err := strconv.ParseInt(notifyChatID, 10, 64)
	if err != nil {
		return
	}
	sendTgMsg(botToken, chatIDInt, text)
}
