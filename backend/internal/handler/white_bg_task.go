package handler

import (
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/integration"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// whiteBgTaskCreateRequest 创建白底图任务请求
type whiteBgTaskCreateRequest struct {
	ImageData    string `json:"image_data" binding:"required"`
	MimeType     string `json:"mime_type"`
	Subject      string `json:"subject"`
	AspectRatio  string `json:"aspect_ratio"`
	Resolution   string `json:"resolution"`
	PreviewImage string `json:"preview_image"` // base64 预览图
	Background   string `json:"background"`
}

// whiteBgTaskResponse 任务响应
type whiteBgTaskResponse struct {
	ID              uint      `json:"id"`
	TaskID          string    `json:"task_id"`
	Status          string    `json:"status"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	Subject         string    `json:"subject"`
	AspectRatio     string    `json:"aspect_ratio"`
	Resolution      string    `json:"resolution"`
	PreviewImageURL string    `json:"preview_image_url"`
	ResultImageURL  string    `json:"result_image_url"`
	IsSavedToDesign bool      `json:"is_saved_to_design"`
	Model           string    `json:"model"`
	Background      string    `json:"background"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateWhiteBgTask 创建白底图任务
func CreateWhiteBgTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	if !acquireRembgTask(c) {
		return
	}
	started := false
	defer func() {
		if !started {
			<-rembgTasks
		}
	}()

	var req whiteBgTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Background == "" {
		req.Background = "white"
	}
	if req.Background != "white" && req.Background != "transparent" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的背景类型"})
		return
	}
	if err := rembgClient().Health(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = "商品抠图"
	}
	aspectRatio := "原图"
	resolution := "原图"
	drawModel := integration.RembgModel

	// 保存预览图
	var previewImageURL string
	if req.PreviewImage != "" {
		previewURL, saveErr := storage.SaveBase64Image(req.PreviewImage, req.MimeType)
		if saveErr == nil {
			previewImageURL = previewURL
		}
	}

	// 创建任务
	taskID := uuid.New().String()
	task := model.WhiteBgTask{
		UserID:          userID,
		TaskID:          taskID,
		Status:          "queued",
		Subject:         subject,
		AspectRatio:     aspectRatio,
		Resolution:      resolution,
		PreviewImageURL: previewImageURL,
		Model:           drawModel,
		Prompt:          "rembg " + req.Background + " background: " + subject,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	// 异步执行任务
	started = true
	go func() {
		defer func() { <-rembgTasks }()
		runWhiteBgTask(taskID, req.ImageData, req.MimeType, subject, aspectRatio, resolution, drawModel, userID, req.Background)
	}()

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "queued",
	})
}

// ListWhiteBgTasks 获取用户白底图任务列表
func ListWhiteBgTasks(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var tasks []model.WhiteBgTask
	query := database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(50)
	if err := query.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败"})
		return
	}

	responses := make([]whiteBgTaskResponse, len(tasks))
	for i, task := range tasks {
		responses[i] = mapWhiteBgTask(task)
	}

	c.JSON(http.StatusOK, gin.H{
		"items": responses,
	})
}

// GetWhiteBgTask 获取单个任务
func GetWhiteBgTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}

	var task model.WhiteBgTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, mapWhiteBgTask(task))
}

// SaveWhiteBgTaskToDesign 保存白底图到设计库
func SaveWhiteBgTaskToDesign(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}

	var task model.WhiteBgTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	if task.ResultImageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务结果不存在"})
		return
	}

	if task.IsSavedToDesign {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片已保存到设计库"})
		return
	}

	now := time.Now()
	if err := database.DB.Model(&task).Updates(map[string]interface{}{
		"is_saved_to_design": true,
		"saved_at":           &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已保存到设计库"})
}

// DeleteWhiteBgTask 删除任务
func DeleteWhiteBgTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务ID不能为空"})
		return
	}

	var task model.WhiteBgTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	if err := database.DB.Delete(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// runWhiteBgTask 异步执行白底图生成任务
func runWhiteBgTask(taskID, imageData, mimeType, subject, aspectRatio, resolution, drawModel string, userID uint, background string) {
	// 更新状态为 running
	database.DB.Model(&model.WhiteBgTask{}).Where("task_id = ?", taskID).Update("status", "running")

	prompt := "rembg " + background + " background: " + subject
	database.DB.Model(&model.WhiteBgTask{}).Where("task_id = ?", taskID).Update("prompt", prompt)
	usageLog := model.UsageLog{
		UserID: userID, UsageType: "image", Source: "white-bg", Model: drawModel,
		Prompt: prompt, Resolution: resolution, Status: "failed",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	ctx, cancel := newImageTaskContext()
	defer cancel()
	imageURL, err := removeBackgroundImage(ctx, imageData, background)
	if err != nil {
		usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
		database.DB.Create(&usageLog)
		database.DB.Model(&model.WhiteBgTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
			"status": "failed", "error_message": err.Error(),
		})
		return
	}

	// 更新任务为完成
	database.DB.Model(&model.WhiteBgTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":           "done",
		"result_image_url": imageURL,
	})

	// 更新 usage log
	usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
	usageLog.Status = "success"
	usageLog.Result = imageURL
	usageLog.ImageURL = imageURL
	usageLog.ReferenceImageURL = resolveUsageReferenceImageURL([]RefImageItem{
		{Data: imageData, MimeType: mimeType},
	}, "", "")
	database.DB.Create(&usageLog)

	go incUsageCount(userID, drawModel)
}

// mapWhiteBgTask 转换为响应格式
func mapWhiteBgTask(task model.WhiteBgTask) whiteBgTaskResponse {
	background := "white"
	if strings.HasPrefix(task.Prompt, "rembg transparent background:") {
		background = "transparent"
	}
	return whiteBgTaskResponse{
		ID:              task.ID,
		TaskID:          task.TaskID,
		Status:          task.Status,
		ErrorMessage:    task.ErrorMessage,
		Subject:         task.Subject,
		AspectRatio:     task.AspectRatio,
		Resolution:      task.Resolution,
		PreviewImageURL: task.PreviewImageURL,
		ResultImageURL:  task.ResultImageURL,
		IsSavedToDesign: task.IsSavedToDesign,
		Model:           task.Model,
		Background:      background,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
}
