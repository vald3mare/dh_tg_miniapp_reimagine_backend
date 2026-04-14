package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type tgLabeledPrice struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"` // в копейках
}

type tgCreateInvoiceLinkReq struct {
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Payload        string           `json:"payload"`
	ProviderToken  string           `json:"provider_token"`
	Currency       string           `json:"currency"`
	Prices         []tgLabeledPrice `json:"prices"`
	PhotoURL       string           `json:"photo_url,omitempty"`
	NeedName       bool             `json:"need_name,omitempty"`
	NeedPhoneNumber bool            `json:"need_phone_number,omitempty"`
}

type tgAPIResponse struct {
	OK     bool   `json:"ok"`
	Result string `json:"result"`
	Description string `json:"description"`
}

// CreateInvoiceLink — POST /payment/invoice
// Создаёт нативную платёжную ссылку Telegram через Bot API.
// Требует TELEGRAM_PROVIDER_TOKEN в env (выдаётся BotFather при подключении ЮKassa).
func CreateInvoiceLink(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			return
		}

		var req struct {
			ItemID uint `json:"item_id" binding:"required,gt=0"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный запрос"})
			return
		}

		botToken := os.Getenv("BOT_TOKEN")
		providerToken := os.Getenv("TELEGRAM_PROVIDER_TOKEN")
		if botToken == "" || providerToken == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Telegram-оплата не настроена: задайте BOT_TOKEN и TELEGRAM_PROVIDER_TOKEN",
			})
			return
		}

		var item models.CatalogItem
		if err := db.First(&item, req.ItemID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Товар не найден"})
			return
		}

		// Telegram принимает цену в наименьших единицах валюты (копейки для RUB)
		amountKopecks := int64(item.Price * 100)

		payload := fmt.Sprintf("user_id=%d|item_id=%d", user.ID, item.ID)

		invoiceReq := tgCreateInvoiceLinkReq{
			Title:         item.Name,
			Description:   item.Description,
			Payload:       payload,
			ProviderToken: providerToken,
			Currency:      "RUB",
			Prices: []tgLabeledPrice{
				{Label: item.Name, Amount: amountKopecks},
			},
		}

		body, _ := json.Marshal(invoiceReq)
		apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/createInvoiceLink", botToken)

		resp, err := http.Post(apiURL, "application/json", bytes.NewReader(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка связи с Telegram API"})
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var tgResp tgAPIResponse
		if err := json.Unmarshal(respBody, &tgResp); err != nil || !tgResp.OK {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "Telegram API вернул ошибку",
				"details": tgResp.Description,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"invoice_url": tgResp.Result})
	}
}

// HandleTelegramPayment — POST /payment/telegram-success
// Вызывается ботом при получении successful_payment от Telegram.
// Создаёт заказ и уведомляет клиента.
func HandleTelegramPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := os.Getenv("ORDERS_API_KEY")
		if apiKey != "" && c.GetHeader("X-API-Key") != apiKey {
			c.JSON(http.StatusForbidden, gin.H{"error": "Неверный API-ключ"})
			return
		}

		var body struct {
			UserID     uint   `json:"user_id"  binding:"required"`
			ItemID     uint   `json:"item_id"  binding:"required"`
			TelegramID uint   `json:"telegram_id"`
			ChargeID   string `json:"charge_id"`
			Amount     int64  `json:"amount"` // в копейках
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var item models.CatalogItem
		if err := db.First(&item, body.ItemID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Товар не найден"})
			return
		}

		var user models.User
		if err := db.First(&user, body.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
			return
		}

		// Сохраняем платёж
		payment := models.Payment{
			UserID:      body.UserID,
			ItemID:      body.ItemID,
			YooKassaID:  "tg-" + body.ChargeID,
			Amount:      float64(body.Amount) / 100,
			Currency:    "RUB",
			Status:      "succeeded",
			Description: fmt.Sprintf("Telegram Pay: %s", item.Name),
		}
		db.Create(&payment)

		// Создаём заказ
		customerName := (user.FirstName + " " + user.LastName)
		order := models.Order{
			CustomerID:  &body.UserID,
			ServiceType: item.Name,
			Description: payment.Description,
			Price:       payment.Amount,
			CustomerName: customerName,
			Status:      "open",
		}
		if err := db.Create(&order).Error; err == nil {
			go NotifyUser(user.TelegramID, fmt.Sprintf(
				"✅ Оплата через Telegram прошла успешно!\n\n"+
					"Услуга: <b>%s</b>\n"+
					"Сумма: %.0f ₽\n\n"+
					"Мы скоро свяжемся с вами 🐾", item.Name, payment.Amount,
			))
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
