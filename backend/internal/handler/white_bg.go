package handler

import (
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/integration"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

// WhiteBgGenerateRequest 白底图生成请求
// 使用 rembg 提取主体并合成白底，保留原图尺寸；Subject 仅作为备注。
type WhiteBgGenerateRequest struct {
	ImageData   string `json:"image_data" binding:"required"`
	MimeType    string `json:"mime_type"`
	Subject     string `json:"subject"`
	AspectRatio string `json:"aspect_ratio"`
	Resolution  string `json:"resolution"`
}

// GenerateWhiteBgImage handles POST /api/white-bg/generate
func GenerateWhiteBgImage(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	if !acquireRembgTask(c) {
		return
	}
	defer func() { <-rembgTasks }()

	var req WhiteBgGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subject := strings.TrimSpace(req.Subject)
	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/png"
	}

	drawModel := integration.RembgModel
	resolution := "原图"
	prompt := "rembg white background: " + subject

	usageLog := model.UsageLog{
		UserID:     userID,
		UsageType:  "image",
		Source:     "white-bg",
		Model:      drawModel,
		Prompt:     prompt,
		Resolution: resolution,
		Status:     "failed",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	imageURL, err := removeBackgroundImage(c.Request.Context(), req.ImageData, "white")
	if err != nil {
		database.DB.Create(&usageLog)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
	usageLog.Status = "success"
	usageLog.Result = imageURL
	usageLog.ImageURL = imageURL
	usageLog.ReferenceImageURL = resolveUsageReferenceImageURL([]RefImageItem{
		{Data: req.ImageData, MimeType: mimeType},
	}, "", "")
	database.DB.Create(&usageLog)
	go incUsageCount(userID, drawModel)

	c.JSON(http.StatusOK, gin.H{
		"image_url": imageURL,
		"mime_type": "image/png",
	})
}
