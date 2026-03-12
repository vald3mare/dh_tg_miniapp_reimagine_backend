package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

// Ключ для хранения parsed initData в контексте
type contextKey string

const (
	_initDataKey contextKey = "init-data"
)

// Сохраняем initData в контекст
func WithInitData(ctx context.Context, initData initdata.InitData) context.Context {
	return context.WithValue(ctx, _initDataKey, initData)
}

// Получаем parsed initData из контекста
func CtxInitData(ctx context.Context) (initdata.InitData, bool) {
	initData, ok := ctx.Value(_initDataKey).(initdata.InitData)
	return initData, ok
}

// AuthMiddleware валидирует заголовок Authorization: tma <initDataRaw>.
// Init data считается валидной в течение 1 часа с момента создания.
func AuthMiddleware(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authParts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(authParts) != 2 {
			c.AbortWithStatusJSON(401, gin.H{"message": "Unauthorized"})
			return
		}

		authType := authParts[0]
		authData := authParts[1]

		switch authType {
		case "tma":
			if err := initdata.Validate(authData, token, time.Hour); err != nil {
				c.AbortWithStatusJSON(401, gin.H{"message": err.Error()})
				return
			}

			initData, err := initdata.Parse(authData)
			if err != nil {
				c.AbortWithStatusJSON(500, gin.H{"message": err.Error()})
				return
			}

			c.Request = c.Request.WithContext(
				WithInitData(c.Request.Context(), initData),
			)
			c.Next()

		default:
			c.AbortWithStatusJSON(401, gin.H{"message": "unsupported auth type"})
		}
	}
}
