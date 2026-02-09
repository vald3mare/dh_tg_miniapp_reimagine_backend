// handlers/payment.go
package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models" // Путь к твоей модели CatalogItem

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreatePaymentRequest struct {
	ItemID uint `json:"item_id" binding:"required"` // ID товара/услуги из БД
}

// Структура для body запроса к YooKassa API (по документации https://yookassa.ru/developers/api#create_payment)
type YooKassaPaymentRequest struct {
	Amount       YooKassaAmount       `json:"amount"`
	Confirmation YooKassaConfirmation `json:"confirmation"`
	Capture      bool                 `json:"capture"`
	Description  string               `json:"description"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
}

type YooKassaAmount struct {
	Value    string `json:"value"` // Строка с двумя знаками после точки, например "1500.00"
	Currency string `json:"currency"`
}

type YooKassaConfirmation struct {
	Type      string `json:"type"` // "redirect"
	ReturnURL string `json:"return_url"`
}

// Структура ответа от YooKassa (упрощённая, нам нужен только confirmation_url)
type YooKassaPaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		Type            string `json:"type"`
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

const (
	yookassaAPIURL = "https://api.yookassa.ru/v3/payments"
	// Тестовые ключи — возьми из личного кабинета YooKassa (раздел "Тестовый магазин")
	testShopID    = "1271879"                                          // Замени на свой тестовый
	testSecretKey = "test_WmGjVYt5HV9ZR9vUqUidTE6H7HXVISNHmKggbRSDqP4" // Замени на свой тестовый
)

func CreateTestPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный запрос"})
			return
		}

		// Находим товар/услугу в БД
		var item models.CatalogItem
		if err := db.First(&item, req.ItemID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Товар/услуга не найдена"})
			return
		}

		// Idempotence-Key — уникальный для каждого запроса
		idempotencyKey := uuid.New().String()

		// Формируем сумму как строку с двумя знаками
		amountValue := fmt.Sprintf("%.2f", item.Price)

		// Описание с явной пометкой о тестовом платеже
		description := fmt.Sprintf("[ТЕСТОВЫЙ ПЛАТЁЖ] Оплата услуги/товара: %s (ID: %d)", item.Name, item.ID)

		// Body запроса
		paymentReq := YooKassaPaymentRequest{
			Amount: YooKassaAmount{
				Value:    amountValue,
				Currency: "RUB",
			},
			Confirmation: YooKassaConfirmation{
				Type:      "redirect",
				ReturnURL: "https://your-frontend-domain.com/payment-success", // Замени на свою страницу успеха (или Telegram deep link)
			},
			Capture:     true, // Автозахват (для теста ок)
			Description: description,
			Metadata: map[string]string{
				"test_payment": "true",
				"item_id":      strconv.FormatUint(uint64(item.ID), 10),
				"user_id":      "test_user", // Потом замени на реальный Telegram ID
			},
		}

		// Marshal в JSON
		bodyBytes, err := json.Marshal(paymentReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка формирования запроса"})
			return
		}

		// Создаём HTTP-запрос
		httpReq, err := http.NewRequestWithContext(context.Background(), "POST", yookassaAPIURL, bytes.NewReader(bodyBytes))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания HTTP-запроса"})
			return
		}

		// Headers
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Idempotence-Key", idempotencyKey)

		// Basic Auth: base64(shopId:secretKey)
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", testShopID, testSecretKey)))
		httpReq.Header.Set("Authorization", "Basic "+auth)

		// Выполняем запрос
		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка связи с YooKassa", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка чтения ответа YooKassa"})
			return
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "YooKassa вернула ошибку",
				"status": resp.StatusCode,
				"body":   string(respBody),
			})
			return
		}

		// Парсим ответ
		var paymentResp YooKassaPaymentResponse
		if err := json.Unmarshal(respBody, &paymentResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка парсинга ответа YooKassa"})
			return
		}

		// Возвращаем фронту URL для оплаты
		c.JSON(http.StatusOK, gin.H{
			"payment_id":       paymentResp.ID,
			"confirmation_url": paymentResp.Confirmation.ConfirmationURL,
			"status":           paymentResp.Status,
			"test_mode":        true,
		})
	}
}
