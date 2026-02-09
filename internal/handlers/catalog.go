package handlers

import (
	"net/http"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

func GetCatalog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []models.CatalogItem

		if err := db.Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить каталог"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
			"total": len(items), // Или отдельный COUNT для пагинации
		})
	}
}
