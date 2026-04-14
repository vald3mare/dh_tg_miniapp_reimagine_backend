package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// bodyWriter перехватывает тело ответа, чтобы залогировать его при ошибке.
type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger — структурированный лог запросов.
//
// Формат успешных:
//
//	→ GET  /profile          200  15ms   user=386696955
//
// Формат ошибок (4xx/5xx):
//
//	→ POST /payment/create   400  5ms    user=386696955  body={"error":"Товар не найден"}
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// OPTIONS-запросы — CORS preflight, не информативны, пропускаем
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		start := time.Now()

		// Оборачиваем writer для перехвата тела ответа
		bw := &bodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = bw

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		// Форматируем латентность
		var latStr string
		switch {
		case latency < time.Millisecond:
			latStr = fmt.Sprintf("%dµs", latency.Microseconds())
		case latency < time.Second:
			latStr = fmt.Sprintf("%dms", latency.Milliseconds())
		default:
			latStr = fmt.Sprintf("%.1fs", latency.Seconds())
		}

		// Достаём telegram_id из контекста (если пользователь авторизован)
		userStr := ""
		if user, ok := CtxUser(c); ok {
			// Маршруты с LoadUser — есть полная модель юзера
			userStr = fmt.Sprintf("  user=%d", user.TelegramID)
		} else if initData, ok := CtxInitData(c.Request.Context()); ok {
			// Маршруты с auth но без LoadUser (например /profile)
			userStr = fmt.Sprintf("  user=%d", initData.User.ID)
		}

		// Цвет статуса
		statusStr := colorStatus(status)

		// Метод — выравниваем до 6 символов
		method := fmt.Sprintf("%-6s", c.Request.Method)

		// Путь + query
		path := c.Request.URL.Path
		if q := c.Request.URL.RawQuery; q != "" {
			path += "?" + q
		}

		// Базовая строка
		ts := time.Now().Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("%s → %s %-35s %s  %-7s%s",
			ts, method, path, statusStr, latStr, userStr)

		// Для ошибок добавляем тело ответа (обрезаем до 300 символов)
		if status >= 400 {
			body := strings.TrimSpace(bw.body.String())
			if body != "" {
				if len(body) > 300 {
					body = body[:300] + "..."
				}
				line += fmt.Sprintf("  → %s", body)
			}
		}

		fmt.Println(line)
	}
}

func colorStatus(status int) string {
	switch {
	case status >= 500:
		return fmt.Sprintf("ERR %d", status)
	case status >= 400:
		return fmt.Sprintf("WRN %d", status)
	case status >= 300:
		return fmt.Sprintf("RDR %d", status)
	default:
		return fmt.Sprintf("OK  %d", status)
	}
}

