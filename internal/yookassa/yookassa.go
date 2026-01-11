package yookassa

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/rvinnie/yookassa-sdk-go/yookassa"
	yoocommon "github.com/rvinnie/yookassa-sdk-go/yookassa/common"
	yoopayment "github.com/rvinnie/yookassa-sdk-go/yookassa/payment"
)

var (
	client         *yookassa.Client
	paymentHandler *yookassa.PaymentHandler
)

// Init инициализирует клиент ЮKassa
func Init() error {
	shopID := os.Getenv("YOOKASSA_SHOP_ID")
	secretKey := os.Getenv("YOOKASSA_SECRET_KEY")

	if shopID == "" || secretKey == "" {
		return fmt.Errorf("не заданы YOOKASSA_SHOP_ID или YOOKASSA_SECRET_KEY")
	}

	client = yookassa.NewClient(shopID, secretKey)
	paymentHandler = yookassa.NewPaymentHandler(client)
	log.Println("ЮKassa клиент инициализирован успешно")
	return nil
}

// CreatePayment создаёт платёж через ЮKassa и возвращает ID платежа и ссылку на оплату
func CreatePayment(ctx context.Context, userID int64, amount float64, description string) (string, string, error) {
	if paymentHandler == nil {
		return "", "", fmt.Errorf("payment handler не инициализирован")
	}

	// Создаём платёж в соответствии с SDK
	// Используем yoocommon.Amount вместо yoopayment.Amount
	payment := &yoopayment.Payment{
		Amount: &yoocommon.Amount{
			Value:    fmt.Sprintf("%.2f", amount),
			Currency: "RUB",
		},
		Confirmation: yoopayment.Redirect{
			Type:      "redirect",
			ReturnURL: os.Getenv("YOOKASSA_RETURN_URL"),
		},
		Description: description,
		Metadata: map[string]interface{}{
			"user_id": fmt.Sprintf("%d", userID),
		},
	}

	// Создаём платёж через обработчик
	resp, err := paymentHandler.CreatePayment(payment)
	if err != nil {
		return "", "", fmt.Errorf("ошибка создания платежа: %w", err)
	}

	if resp == nil {
		return "", "", fmt.Errorf("пустой ответ от сервера")
	}

	// Извлекаем URL подтверждения
	var confirmationURL string
	if resp.Confirmation != nil {
		// Проверяем тип подтверждения
		switch conf := resp.Confirmation.(type) {
		case yoopayment.Redirect:
			confirmationURL = conf.ReturnURL
		default:
			log.Printf("Неподдерживаемый тип подтверждения: %T", resp.Confirmation)
		}
	}

	if confirmationURL == "" {
		return "", "", fmt.Errorf("не получена ссылка на оплату")
	}

	log.Printf("Платёж создан успешно: ID=%s", resp.ID)
	return resp.ID, confirmationURL, nil
}

// HandleWebhook обрабатывает вебхук от ЮKassa
func HandleWebhook(ctx context.Context, payload []byte) error {
	if paymentHandler == nil {
		return fmt.Errorf("payment handler не инициализирован")
	}

	// Парсим JSON вебхука в объект Payment напрямую
	var webhookPayment yoopayment.Payment
	if err := json.Unmarshal(payload, &webhookPayment); err != nil {
		return fmt.Errorf("ошибка парсинга webhook: %w", err)
	}

	// Логируем полученное событие
	log.Printf("Webhook получен: payment_id=%s, status=%s", webhookPayment.ID, webhookPayment.Status)

	// Проверяем статус платежа
	if webhookPayment.Status != yoopayment.Succeeded {
		log.Printf("Webhook: платёж %s имеет статус %s, игнорируем", webhookPayment.ID, webhookPayment.Status)
		return nil
	}

	// Извлекаем user_id из metadata
	var userID int64
	if webhookPayment.Metadata != nil {
		// Metadata может быть map[string]interface{}
		if metadataMap, ok := webhookPayment.Metadata.(map[string]interface{}); ok {
			if userIDVal, ok := metadataMap["user_id"]; ok {
				if userIDStr, ok := userIDVal.(string); ok {
					if parsedID, err := parseUserID(userIDStr); err == nil {
						userID = parsedID
					} else {
						return fmt.Errorf("ошибка парсинга user_id из metadata: %w", err)
					}
				}
			}
		}
	}

	if userID == 0 {
		return fmt.Errorf("не найден user_id в metadata платежа")
	}

	log.Printf("Webhook: обработан платёж ID=%s для пользователя %d", webhookPayment.ID, userID)

	// TODO: Здесь нужна логика активации подписки в БД
	// activateSubscription(ctx, userID, webhookPayment.ID)

	return nil
}

// GetPayment получает информацию о платеже
func GetPayment(ctx context.Context, paymentID string) (*yoopayment.Payment, error) {
	if paymentHandler == nil {
		return nil, fmt.Errorf("payment handler не инициализирован")
	}

	payment, err := paymentHandler.FindPayment(paymentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о платеже: %w", err)
	}

	return payment, nil
}

// CancelPayment отменяет платёж
func CancelPayment(ctx context.Context, paymentID string) (*yoopayment.Payment, error) {
	if paymentHandler == nil {
		return nil, fmt.Errorf("payment handler не инициализирован")
	}

	payment, err := paymentHandler.CancelPayment(paymentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка отмены платежа: %w", err)
	}

	return payment, nil
}

// CapturePayment подтверждает платёж (используется для платежей в статусе waiting_for_capture)
func CapturePayment(ctx context.Context, paymentID string) (*yoopayment.Payment, error) {
	if paymentHandler == nil {
		return nil, fmt.Errorf("payment handler не инициализирован")
	}

	// CapturePayment принимает Payment объект, поэтому сначала получаем платёж
	payment, err := GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	capturedPayment, err := paymentHandler.CapturePayment(payment)
	if err != nil {
		return nil, fmt.Errorf("ошибка подтверждения платежа: %w", err)
	}

	return capturedPayment, nil
}

// parseUserID парсит ID пользователя из строки
func parseUserID(userIDStr string) (int64, error) {
	var userID int64
	_, err := fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}
