package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/Vald3mare/dogshappinies/backend_reimagine/internal/models"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// catalogCache кэширует список услуг на catalogCacheTTL секунд.
// Инвалидируется при создании/обновлении/удалении через Admin API.
var catalogCache struct {
	sync.RWMutex
	items     []models.CatalogItem
	total     int64
	expiresAt time.Time
}

const catalogCacheTTL = 60 * time.Second

// InvalidateCatalogCache сбрасывает кэш каталога.
// Вызывается из Admin-хендлеров после изменения каталога.
func InvalidateCatalogCache() {
	catalogCache.Lock()
	catalogCache.expiresAt = time.Time{}
	catalogCache.Unlock()
}

func GetCatalog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := parsePagination(c)

		// Для пагинированных запросов не отдаём из кэша — кэш только для первой страницы
		useCache := offset == 0 && limit == 20

		if useCache {
			catalogCache.RLock()
			if time.Now().Before(catalogCache.expiresAt) {
				items := catalogCache.items
				total := catalogCache.total
				catalogCache.RUnlock()
				c.JSON(http.StatusOK, gin.H{
					"items":    items,
					"total":    total,
					"has_more": int64(limit) < total,
					"cached":   true,
				})
				return
			}
			catalogCache.RUnlock()
		}

		var total int64
		db.Model(&models.CatalogItem{}).Count(&total)

		var items []models.CatalogItem
		if err := db.Limit(limit).Offset(offset).Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить каталог"})
			return
		}

		if useCache {
			catalogCache.Lock()
			catalogCache.items = items
			catalogCache.total = total
			catalogCache.expiresAt = time.Now().Add(catalogCacheTTL)
			catalogCache.Unlock()
		}

		c.JSON(http.StatusOK, gin.H{
			"items":    items,
			"total":    total,
			"has_more": int64(offset+limit) < total,
		})
	}
}
