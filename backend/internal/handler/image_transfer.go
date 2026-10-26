package handler

import (
	"net/http"
	"strings"

	"advertising-image/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ensureImageOSSRequest struct {
	ImageURL string `json:"imageUrl" binding:"required"`
}

func EnsureImageOSS(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req ensureImageOSSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imageURL := strings.TrimSpace(req.ImageURL)
	if imageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片地址为空"})
		return
	}

	result, err := service.EnsureUserImageOSS(userID, imageURL)
	if err != nil {
		if err == service.ErrImageNotOwned {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权转存该图片"})
			return
		}
		service.LogTransferFailure("ensure user image", imageURL, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "图片转存失败，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"imageUrl":    result.ImageURL,
		"transferred": result.Transferred,
		"updated":     result.Updated,
	})
}
