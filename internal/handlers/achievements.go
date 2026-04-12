package handlers

import (
	"fmt"
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/middleware"
	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAchievements — GET /executor/achievements, защищённый
func GetAchievements(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CtxUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка авторизации"})
			return
		}

		var allAchievements []models.Achievement
		if err := db.Find(&allAchievements).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить ачивки"})
			return
		}

		var userAchievements []models.UserAchievement
		if err := db.Where("user_id = ?", user.ID).Preload("Achievement").Find(&userAchievements).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить ачивки пользователя"})
			return
		}

		earnedMap := make(map[uint]models.UserAchievement)
		for _, ua := range userAchievements {
			earnedMap[ua.AchievementID] = ua
		}

		type AchievementResponse struct {
			models.Achievement
			Earned   bool   `json:"earned"`
			EarnedAt string `json:"earned_at,omitempty"`
		}

		result := make([]AchievementResponse, 0, len(allAchievements))
		for _, a := range allAchievements {
			ar := AchievementResponse{Achievement: a}
			if ua, ok := earnedMap[a.ID]; ok {
				ar.Earned = true
				ar.EarnedAt = ua.CreatedAt.Format("2006-01-02")
			}
			result = append(result, ar)
		}

		c.JSON(http.StatusOK, gin.H{
			"achievements":     result,
			"orders_completed": user.OrdersCompleted,
			"rating":           user.Rating,
		})
	}
}

// CheckAndGrantAchievements — вызывается после завершения заказа.
// Проверяет threshold-ачивки и выдаёт те, которых ещё нет у пользователя.
// Уведомляет пользователя через бота о каждой новой ачивке.
func CheckAndGrantAchievements(db *gorm.DB, user *models.User) {
	var allAchievements []models.Achievement
	db.Where("condition_type = ?", "orders_completed").Find(&allAchievements)

	for _, a := range allAchievements {
		if user.OrdersCompleted < a.Threshold {
			continue
		}
		var count int64
		db.Model(&models.UserAchievement{}).
			Where("user_id = ? AND achievement_id = ?", user.ID, a.ID).
			Count(&count)
		if count > 0 {
			continue
		}
		db.Create(&models.UserAchievement{
			UserID:        user.ID,
			AchievementID: a.ID,
		})
		go NotifyUser(user.TelegramID, fmt.Sprintf(
			"🏆 <b>Новое достижение!</b>\n\n%s <b>%s</b>\n%s",
			a.IconEmoji, a.Name, a.Description,
		))
	}
}
