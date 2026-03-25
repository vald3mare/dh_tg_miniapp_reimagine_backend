package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// parsePagination читает ?limit=N&offset=M из query-параметров.
// Допустимый диапазон limit: 1–100, по умолчанию 20.
// offset >= 0, по умолчанию 0.
func parsePagination(c *gin.Context) (limit, offset int) {
	limit = 20
	offset = 0
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && o >= 0 {
		offset = o
	}
	return
}
