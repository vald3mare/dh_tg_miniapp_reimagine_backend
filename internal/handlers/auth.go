package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ShowInitData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "ShowInitData handler",
	})
}
