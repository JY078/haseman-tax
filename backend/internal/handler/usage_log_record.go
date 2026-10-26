package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type recordUsageLogDownloadRequest struct {
	ImageURL    string `json:"image_url" binding:"required"`
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Resolution  string `json:"resolution"`
	AspectRatio string `json:"aspect_ratio"`
	Source      string `json:"source"`
	TaskID      string `json:"task_id"`
}

func RecordUsageLogDownload(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req recordUsageLogDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	imageURL := strings.TrimSpace(req.ImageURL)
	if imageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_url is required"})
		return
	}

	var logItem model.UsageLog
	err := database.DB.Where("user_id = ? AND image_url = ? AND status = ?", userID, imageURL, "success").
		Order("created_at DESC").
		First(&logItem).Error
	if err == nil {
		if updateErr := database.DB.Model(&model.UsageLog{}).
			Where("id = ?", logItem.ID).
			UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error; updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update download count"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": logItem.ID})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query usage log"})
		return
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "workflow_canvas"
	}
	resolution := strings.TrimSpace(req.Resolution)
	if resolution == "" {
		resolution = strings.TrimSpace(req.AspectRatio)
	}
	result := "[图片已生成]"
	if taskID := strings.TrimSpace(req.TaskID); taskID != "" {
		result = "task:" + taskID
	}

	logItem = model.UsageLog{
		UserID:        userID,
		UsageType:     "drawing",
		Source:        source,
		Model:         strings.TrimSpace(req.Model),
		Prompt:        strings.TrimSpace(req.Prompt),
		Result:        result,
		ImageURL:      imageURL,
		Resolution:    resolution,
		Status:        "success",
		DownloadCount: 1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if logItem.Model == "" {
		logItem.Model = "workflow-canvas"
	}
	if err := database.DB.Create(&logItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create usage log"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": logItem.ID})
}
