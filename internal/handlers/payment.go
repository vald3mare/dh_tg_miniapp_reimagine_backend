package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Типы запроса/ответа ──────────────────────────────────────────────────────

type CreatePaymentRequest struct {
	ItemID uint `json:"item_id" binding:"required,gt=0"`
}

type yooPaymentRequest struct {
	Amount       yooAmount         `json:"amount"`
	Confirmation yooConfirmation   `json:"confirmation"`
	Capture      bool              `json:"capture"`
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Receipt      *yooReceipt       `json:"receipt,omitempty"`
}

type yooAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type yooConfirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

// yooReceipt — данные онлайн-чека (54-ФЗ).
// Обязательны для продакшн-магазинов с онлайн-кассой.
// Заполняется если задана переменная RECEIPT_EMAIL в env.
type yooReceipt struct {
	Customer yooReceiptCustomer `json:"customer"`
	Items    []yooReceiptItem   `json:"items"`
}

type yooReceiptCustomer struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

type yooReceiptItem struct {
	Description    string    `json:"description"`
	Quantity       string    `json:"quantity"`
	Amount         yooAmount `json:"amount"`
	VatCode        int       `json:"vat_code"`        // 1 = без НДС
	PaymentMode    string    `json:"payment_mode"`    // full_payment
	PaymentSubject string    `json:"payment_subject"` // service
}

type yooPaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

// Webhook payload от ЮKassa
type webhookPayload struct {
	Type   string `json:"type"`
	Event  string `json:"event"`
	Object struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"object"`
}

const yookassaAPIURL = "https://api.yookassa.ru/v3/payments"

// IP-диапазоны серверов ЮKassa (https://yookassa.ru/developers/using-api/webhooks)
var yookassaCIDRs = []string{
	"185.71.76.0/27",
	"185.71.77.0/27",
	"77.75.153.0/25",
	"77.75.154.128/25",
	"2a02:5180::/32",
}

// isAllowedWebhookIP проверяет, входит ли IP в диапазоны ЮKassa.
func isAllowedWebhookIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range yookassaCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(parsed) {
			return true
		}
	}
	return false
}

// ─── Хендлеры ─────────────────────────────────────────────────────────────────

// CreatePayment создаёт платёж в ЮKassa, сохраняет запись в БД и возвращает
// ссылку на оплату фронтенду. Маршрут защищён — требует Authorization: tma ...
func CreatePayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			return
		}

		var req CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный запрос"})
			return
		}

		var item models.CatalogItem
		if err := db.First(&item, req.ItemID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Товар/услуга не найдена"})
			return
		}

		shopID := os.Getenv("YOOKASSA_SHOP_ID")
		secretKey := os.Getenv("YOOKASSA_SECRET_KEY")
		if shopID == "" || secretKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Платёжный сервис не настроен"})
			return
		}

		returnURL := os.Getenv("PAYMENT_RETURN_URL")
		if returnURL == "" {
			returnURL = "https://t.me/"
		}

		description := fmt.Sprintf("Оплата услуги: %s (ID: %d)", item.Name, item.ID)
		idempotencyKey := uuid.New().String()

		amountStr := fmt.Sprintf("%.2f", item.Price)

		paymentReq := yooPaymentRequest{
			Amount: yooAmount{
				Value:    amountStr,
				Currency: "RUB",
			},
			Confirmation: yooConfirmation{
				Type:      "redirect",
				ReturnURL: returnURL,
			},
			Capture:     true,
			Description: description,
			Metadata: map[string]string{
				"user_id": strconv.FormatUint(uint64(user.ID), 10),
				"item_id": strconv.FormatUint(uint64(item.ID), 10),
			},
		}

		// Добавляем чек если задан RECEIPT_EMAIL (обязательно для продакшн-магазина с онлайн-кассой)
		if receiptEmail := os.Getenv("RECEIPT_EMAIL"); receiptEmail != "" {
			paymentReq.Receipt = &yooReceipt{
				Customer: yooReceiptCustomer{Email: receiptEmail},
				Items: []yooReceiptItem{
					{
						Description:    item.Name,
						Quantity:       "1.00",
						Amount:         yooAmount{Value: amountStr, Currency: "RUB"},
						VatCode:        1, // 1 = без НДС
						PaymentMode:    "full_payment",
						PaymentSubject: "service",
					},
				},
			}
		}

		bodyBytes, err := json.Marshal(paymentReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка формирования запроса"})
			return
		}

		httpReq, err := http.NewRequestWithContext(context.Background(), "POST", yookassaAPIURL, bytes.NewReader(bodyBytes))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания HTTP-запроса"})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Idempotence-Key", idempotencyKey)
		httpReq.Header.Set("Authorization", "Basic "+
			base64.StdEncoding.EncodeToString([]byte(shopID+":"+secretKey)))

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка связи с ЮKassa", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка чтения ответа ЮKassa"})
			return
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "ЮKassa вернула ошибку",
				"status": resp.StatusCode,
				"body":   string(respBody),
			})
			return
		}

		var yooResp yooPaymentResponse
		if err := json.Unmarshal(respBody, &yooResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка парсинга ответа ЮKassa"})
			return
		}

		payment := models.Payment{
			UserID:      user.ID,
			ItemID:      item.ID,
			YooKassaID:  yooResp.ID,
			Amount:      item.Price,
			Currency:    "RUB",
			Status:      yooResp.Status,
			Description: description,
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Printf("WARN: не удалось сохранить платёж %s в БД: %v", yooResp.ID, err)
		}

		c.JSON(http.StatusOK, gin.H{
			"payment_id":       yooResp.ID,
			"confirmation_url": yooResp.Confirmation.ConfirmationURL,
			"status":           yooResp.Status,
		})
	}
}

// HandlePaymentWebhook обрабатывает уведомления от ЮKassa об изменении статуса платежа.
// В production (GIN_MODE=release) принимает запросы только с IP-адресов ЮKassa.
func HandlePaymentWebhook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем IP только в production (GIN_MODE=release)
		if gin.Mode() == gin.ReleaseMode {
			clientIP := c.ClientIP()
			if !isAllowedWebhookIP(clientIP) {
				log.Printf("Webhook: отклонён запрос с IP %s", clientIP)
				c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
				return
			}
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось прочитать тело запроса"})
			return
		}

		var payload webhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат webhook"})
			return
		}

		log.Printf("Webhook: event=%s payment_id=%s status=%s",
			payload.Event, payload.Object.ID, payload.Object.Status)

		var payment models.Payment
		if err := db.Where("yoo_kassa_id = ?", payload.Object.ID).First(&payment).Error; err != nil {
			log.Printf("Webhook: платёж %s не найден в БД", payload.Object.ID)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}

		if err := db.Model(&payment).Update("status", payload.Object.Status).Error; err != nil {
			log.Printf("Webhook: не удалось обновить статус платежа %s: %v", payment.YooKassaID, err)
		}

		if payload.Object.Status == "succeeded" {
			log.Printf("Webhook: платёж %s успешно оплачен (user_id=%d, item_id=%d)",
				payment.YooKassaID, payment.UserID, payment.ItemID)

			// Активируем подписку, если тип товара — subscription
			var item models.CatalogItem
			if err := db.First(&item, payment.ItemID).Error; err == nil && item.Type == "subscription" {
				now := time.Now()
				sub := models.Subscription{
					UserID:    &payment.UserID,
					Plan:      item.Name,
					Active:    true,
					StartDate: now,
					EndDate:   now.AddDate(0, 1, 0),
					PaymentID: payment.YooKassaID,
				}
				if err := db.Where(models.Subscription{UserID: &payment.UserID}).
					Assign(models.Subscription{
						Plan:      sub.Plan,
						Active:    sub.Active,
						StartDate: sub.StartDate,
						EndDate:   sub.EndDate,
						PaymentID: sub.PaymentID,
					}).
					FirstOrCreate(&sub).Error; err != nil {
					log.Printf("Webhook: не удалось активировать подписку для user_id=%d: %v", payment.UserID, err)
				} else {
					log.Printf("Webhook: подписка активирована для user_id=%d до %s", payment.UserID, sub.EndDate.Format("2006-01-02"))
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
