package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

// Ключ для хранения parsed initData в контексте
type contextKey string

const _initDataKey contextKey = "init-data"

// Сохраняем parsed initData в контекст
func WithInitData(ctx context.Context, initData initdata.InitData) context.Context {
	return context.WithValue(ctx, _initDataKey, initData)
}

// Получаем parsed initData из контекста
func CtxInitData(ctx context.Context) (initdata.InitData, bool) {
	val, ok := ctx.Value(_initDataKey).(initdata.InitData)
	return val, ok
}

// Middleware авторизации по заголовку Authorization: tma <initDataRaw>
func AuthMiddleware(botToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("Missing Authorization header from %s", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Missing Authorization header"})
			return
		}

		authParts := strings.SplitN(authHeader, " ", 2)
		if len(authParts) != 2 || authParts[0] != "tma" {
			log.Printf("Invalid Authorization format from %s", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid Authorization format. Expected: tma <initData>"})
			return
		}

		rawInitData := authParts[1]
		log.Println("Raw init data:", rawInitData)

		// Валидируем подпись (expIn = 0 = без проверки времени жизни)
		// Это важно в разработке где initData может быть старой
		if err := initdata.Validate(rawInitData, botToken, 0); err != nil {
			log.Printf("Validation failed from %s: %v", c.ClientIP(), err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid initData: " + err.Error()})
			return
		}

		// Парсим initData
		parsed, err := initdata.Parse(rawInitData)
		if err != nil {
			log.Printf("Parse failed from %s: %v", c.ClientIP(), err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse initData: " + err.Error()})
			return
		}

		// Сохраняем в контекст
		c.Request = c.Request.WithContext(WithInitData(c.Request.Context(), parsed))

		// Логируем успешную авторизацию
		log.Printf("Auth success: UserID=%d Username=%s IP=%s", parsed.User.ID, parsed.User.Username, c.ClientIP())

		c.Next()
	}
}
