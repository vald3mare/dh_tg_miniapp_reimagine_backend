package handlers

import (
	"net/http"
	"os"
	"strconv"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SubmitApplication — POST /bot/application
// Вызывается Telegram-ботом после завершения анкеты исполнителя.
// Защищён заголовком X-Bot-Key == BOT_API_KEY из env.
func SubmitApplication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Переиспользуем ORDERS_API_KEY — тот же паттерн «доверенный внутренний вызов».
		// Если ORDERS_AUTH_DISABLED=true, ключ не проверяется.
		apiKey := os.Getenv("ORDERS_API_KEY")
		authDisabled := os.Getenv("ORDERS_AUTH_DISABLED") == "true"
		if !authDisabled && (apiKey == "" || c.GetHeader("X-API-Key") != apiKey) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Неверный или отсутствующий API-ключ"})
			return
		}

		var body struct {
			TelegramID       uint   `json:"telegram_id"       binding:"required"`
			TelegramUsername string `json:"telegram_username"`
			FullName         string `json:"full_name"`
			FormDataJSON     string `json:"form_data_json"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		app := models.ExecutorApplication{
			TelegramID:       body.TelegramID,
			TelegramUsername: body.TelegramUsername,
			FullName:         body.FullName,
			FormDataJSON:     body.FormDataJSON,
			Status:           "pending",
		}
		if err := db.Create(&app).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить заявку"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": app.ID, "status": app.Status})
	}
}

// AdminListApplications — GET /admin/applications
// Поддерживает фильтр ?status=pending|approved|rejected и пагинацию.
func AdminListApplications(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		limit, offset := parsePagination(c)

		q := db.Model(&models.ExecutorApplication{})
		if status != "" {
			q = q.Where("status = ?", status)
		}

		var total int64
		q.Count(&total)

		var apps []models.ExecutorApplication
		if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&apps).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить заявки"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"applications": apps,
			"total":        total,
			"has_more":     int64(offset+limit) < total,
		})
	}
}

// AdminApproveApplication — POST /admin/applications/:id/approve
// Одобряет заявку, выдаёт роль executor пользователю и уведомляет его в боте.
func AdminApproveApplication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		var app models.ExecutorApplication
		if err := db.First(&app, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
			return
		}
		if app.Status == "approved" {
			c.JSON(http.StatusConflict, gin.H{"error": "Заявка уже одобрена"})
			return
		}

		// Обновляем статус заявки
		app.Status = "approved"
		app.ReviewedBy = &admin.ID
		if err := db.Save(&app).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить заявку"})
			return
		}

		// Выдаём роль executor: ищем пользователя по TelegramID
		var user models.User
		if db.Where("telegram_id = ?", app.TelegramID).First(&user).Error == nil {
			// Пользователь уже есть в базе — добавляем роль
			if !containsStr(user.Roles, "executor") {
				user.Roles = append(user.Roles, "executor")
			}
			user.Role = "executor"
			db.Save(&user)
		} else {
			// Пользователь ещё не открывал мини-апп — создаём запись заранее
			newUser := models.User{
				TelegramID: app.TelegramID,
				Username:   app.TelegramUsername,
				FirstName:  app.FullName,
				Role:       "executor",
				Roles:      []string{"executor"},
			}
			db.Create(&newUser)
		}

		// Уведомляем кандидата через бота
		go NotifyUser(app.TelegramID,
			"🎉 <b>Поздравляем!</b>\n\n"+
				"Ваша заявка на роль исполнителя одобрена. "+
				"Теперь вы можете принимать заказы в приложении <b>Собачье Счастье</b>! 🐾")

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// AdminRejectApplication — POST /admin/applications/:id/reject
// Отклоняет заявку с опциональным комментарием и уведомляет кандидата.
func AdminRejectApplication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
			return
		}

		var body struct {
			Note string `json:"note"`
		}
		_ = c.ShouldBindJSON(&body)

		var app models.ExecutorApplication
		if err := db.First(&app, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
			return
		}

		app.Status = "rejected"
		app.ReviewNote = body.Note
		app.ReviewedBy = &admin.ID
		if err := db.Save(&app).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить заявку"})
			return
		}

		msg := "😔 К сожалению, ваша заявка на роль исполнителя была отклонена."
		if body.Note != "" {
			msg += "\n\n💬 <b>Комментарий:</b> " + body.Note
		}
		go NotifyUser(app.TelegramID, msg)

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
