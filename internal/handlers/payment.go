package handlers

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/yookassa"
	"github.com/gin-gonic/gin"
)

// CreatePaymentRequest содержит данные для создания платежа
type CreatePaymentRequest struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description" binding:"required"`
}

// CreatePaymentResponse содержит результат создания платежа
type CreatePaymentResponse struct {
	PaymentID       string `json:"payment_id"`
	ConfirmationURL string `json:"confirmation_url"`
}

// CreatePayment создаёт новый платёж
func CreatePayment(c *gin.Context) {
	ctx := c.Request.Context()

	// Получаем данные инициализации из контекста (заполнено middleware)
	initData, ok := middleware.CtxInitData(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Init data not found"})
		return
	}

	userID := initData.User.ID

	// Парсим request body
	var req CreatePaymentRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Создаём платёж через ЮKassa
	paymentID, confirmationURL, err := yookassa.CreatePayment(ctx, userID, req.Amount, req.Description)
	if err != nil {
		log.Printf("Failed to create payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
		return
	}

	c.JSON(http.StatusOK, CreatePaymentResponse{
		PaymentID:       paymentID,
		ConfirmationURL: confirmationURL,
	})
}

// GetPaymentStatus получает статус платежа
func GetPaymentStatus(c *gin.Context) {
	ctx := c.Request.Context()

	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id is required"})
		return
	}

	payment, err := yookassa.GetPayment(ctx, paymentID)
	if err != nil {
		log.Printf("Failed to get payment status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_id": payment.ID,
		"status":     payment.Status,
		"amount":     payment.Amount,
		"created_at": payment.CreatedAt,
	})
}

// YookassaWebhook обрабатывает вебхуки от ЮKassa
// Важно: вебхук должен быть открыт (без авторизации)
func YookassaWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	// Читаем тело запроса
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	log.Printf("Received webhook: %s", string(body))

	// Обрабатываем вебхук
	if err := yookassa.HandleWebhook(ctx, body); err != nil {
		log.Printf("Failed to handle webhook: %v", err)
		// ВАЖНО: Даже при ошибке возвращаем 200 OK, чтобы ЮKassa не повторял отправку
		c.JSON(http.StatusOK, gin.H{"status": "received"})
		return
	}

	// Всегда возвращаем 200 OK для подтверждения получения вебхука
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CancelPayment отменяет платёж
func CancelPayment(c *gin.Context) {
	ctx := c.Request.Context()

	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id is required"})
		return
	}

	// Проверяем, что платёж принадлежит текущему пользователю
	initData, ok := middleware.CtxInitData(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Init data not found"})
		return
	}

	// TODO: Добавить проверку что платёж принадлежит пользователю
	_ = initData.User.ID

	payment, err := yookassa.CancelPayment(ctx, paymentID)
	if err != nil {
		log.Printf("Failed to cancel payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_id": payment.ID,
		"status":     payment.Status,
	})
}

// CapturePayment подтверждает платёж (для платежей в статусе waiting_for_capture)
func CapturePayment(c *gin.Context) {
	ctx := c.Request.Context()

	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id is required"})
		return
	}

	// Проверяем, что платёж принадлежит текущему пользователю
	initData, ok := middleware.CtxInitData(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Init data not found"})
		return
	}

	userID := initData.User.ID
	_ = userID // TODO: Использовать для проверки принадлежности платежа

	payment, err := yookassa.CapturePayment(ctx, paymentID)
	if err != nil {
		log.Printf("Failed to capture payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to capture payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_id": payment.ID,
		"status":     payment.Status,
	})
}

// PaymentSuccess обрабатывает редирект от ЮKassa после оплаты
// и редиректит пользователя обратно в Mini App
func PaymentSuccess(c *gin.Context) {
	paymentID := c.Query("payment_id")

	// Опционально: проверить статус платежа в БД
	if paymentID != "" {
		log.Printf("Payment success redirect for payment_id: %s", paymentID)
	}

	// Получаем URL Mini App из env
	miniAppURL := os.Getenv("TELEGRAM_MINIAPP_URL")
	if miniAppURL == "" {
		miniAppURL = "https://t.me/dogs_happiness_bot/miniapp"
	}

	// Редиректим в Mini App с параметром success
	c.Redirect(http.StatusFound, miniAppURL+"?startapp=payment_success")
}
