package handler

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxDepthImageCount = 6

type DepthImageInput struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
}

type CreateDepthTaskRequest struct {
	Images      []DepthImageInput `json:"images" binding:"required"`
	AspectRatio string            `json:"aspect_ratio"`
	Quality     string            `json:"quality"`
}

func CreateDepthTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req CreateDepthTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}
	if len(req.Images) == 0 || len(req.Images) > maxDepthImageCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 1-6 张图片"})
		return
	}
	if req.AspectRatio == "" {
		req.AspectRatio = "1:1"
	}
	if req.Quality != "2K" && req.Quality != "4K" {
		req.Quality = "2K"
	}

	taskID := uuid.New().String()
	task := model.DepthTask{
		UserID:      userID,
		TaskID:      taskID,
		Status:      "queued",
		AspectRatio: req.AspectRatio,
		Quality:     req.Quality,
		ImageCount:  len(req.Images),
	}

	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}
	for i, img := range req.Images {
		refURL := ""
		if strings.TrimSpace(img.Data) != "" {
			if url, err := storage.SavePromptImage(img.Data, firstNonEmpty(img.MimeType, "image/png")); err == nil {
				refURL = url
			}
		}
		database.DB.Create(&model.DepthImage{
			TaskIDRef:         taskID,
			Sort:              i,
			ReferenceImageURL: refURL,
		})
	}

	go runDepthTask(taskID, userID, req)

	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
}

func buildDepthPrompt() string {
	return "将图片转为景深图"
}

func runDepthTask(taskID string, userID uint, req CreateDepthTaskRequest) {
	database.DB.Model(&model.DepthTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":     "running",
		"updated_at": time.Now(),
	})

	apiKey := getSettingValue("draw_key")
	if apiKey == "" {
		failDepthTask(taskID, "Drawing API key not configured")
		return
	}
	modelName := getDefaultModelByType("drawing", "")
	if modelName == "" {
		failDepthTask(taskID, "未配置默认绘图模型")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	prompt := buildDepthPrompt()
	var wg sync.WaitGroup
	hasError := false
	var mu sync.Mutex

	for i, input := range req.Images {
		wg.Add(1)
		go func(sortIndex int, img DepthImageInput) {
			defer wg.Done()
			startedAt := time.Now()
			var depthImage model.DepthImage
			database.DB.Where("task_id_ref = ? AND sort = ?", taskID, sortIndex).First(&depthImage)
			imageURL, err := generateDepthImage(ctx, apiKey, modelName, prompt, img, req.AspectRatio, req.Quality)
			update := map[string]interface{}{"updated_at": time.Now()}
			if err != nil {
				update["error_msg"] = err.Error()
				mu.Lock()
				hasError = true
				mu.Unlock()
			} else {
				update["image_url"] = imageURL
				usageID := createDepthUsageLog(userID, modelName, prompt, req.Quality, req.AspectRatio, imageURL, depthImage.ReferenceImageURL, sortIndex, int(time.Since(startedAt).Milliseconds()))
				update["usage_log_id"] = usageID
			}
			database.DB.Model(&model.DepthImage{}).Where("task_id_ref = ? AND sort = ?", taskID, sortIndex).Updates(update)
		}(i, input)
	}
	wg.Wait()

	status := "done"
	if hasError {
		status = "failed"
	}
	database.DB.Model(&model.DepthTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})
}

func generateDepthImage(ctx context.Context, apiKey, modelName, prompt string, img DepthImageInput, aspectRatio, quality string) (string, error) {
	if strings.TrimSpace(img.Data) == "" {
		return "", fmt.Errorf("参考图数据为空")
	}
	return generateProductReplaceWithRefs(ctx, apiKey, modelName, prompt, []adapter.RefImage{{
		Data:     img.Data,
		MimeType: firstNonEmpty(img.MimeType, "image/png"),
	}}, aspectRatio, quality)
}

func createDepthUsageLog(userID uint, modelName, prompt, quality, aspectRatio, imageURL, referenceImageURL string, sortIndex int, duration int) uint {
	logItem := model.UsageLog{
		UserID:            userID,
		UsageType:         "drawing",
		Source:            "depth_of_field",
		Model:             modelName,
		Prompt:            prompt,
		Result:            "[景深图已生成]",
		ImageURL:          imageURL,
		ReferenceImageURL: referenceImageURL,
		Resolution:        quality,
		RequestBody:       fmt.Sprintf(`{"sort":%d,"quality":"%s","aspect_ratio":"%s"}`, sortIndex, quality, aspectRatio),
		Status:            "success",
		Duration:          duration,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	database.DB.Create(&logItem)
	go incUsageCount(userID, modelName)
	return logItem.ID
}

func failDepthTask(taskID string, errMsg string) {
	database.DB.Model(&model.DepthImage{}).Where("task_id_ref = ? AND image_url = ''", taskID).Updates(map[string]interface{}{
		"error_msg":  errMsg,
		"updated_at": time.Now(),
	})
	database.DB.Model(&model.DepthTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": errMsg,
		"updated_at":    time.Now(),
	})
}

func GetDepthTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("id")

	var task model.DepthTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Images").First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	sort.Slice(task.Images, func(i, j int) bool { return task.Images[i].Sort < task.Images[j].Sort })

	images := make([]gin.H, 0, len(task.Images))
	for _, img := range task.Images {
		status := "pending"
		if img.ImageURL != "" {
			status = "done"
		} else if img.ErrorMsg != "" {
			status = "error"
		} else if task.Status == "running" {
			status = "generating"
		}
		images = append(images, gin.H{
			"sort":         img.Sort,
			"image_url":    img.ImageURL,
			"error":        img.ErrorMsg,
			"status":       status,
			"usage_log_id": img.UsageLogID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": task.TaskID,
		"status":  task.Status,
		"images":  images,
		"error":   task.ErrorMessage,
	})
}

func ListDepthTasks(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 20 {
		pageSize = 10
	}

	var total int64
	database.DB.Model(&model.DepthTask{}).Where("user_id = ?", userID).Count(&total)

	var tasks []model.DepthTask
	if err := database.DB.Where("user_id = ?", userID).
		Preload("Images").
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败"})
		return
	}

	result := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		doneCount := 0
		failedCount := 0
		cover := ""
		images := make([]gin.H, 0, len(task.Images))
		sort.Slice(task.Images, func(i, j int) bool { return task.Images[i].Sort < task.Images[j].Sort })
		for _, img := range task.Images {
			if img.ImageURL != "" {
				doneCount++
				if cover == "" {
					cover = img.ImageURL
				}
			} else if img.ErrorMsg != "" {
				failedCount++
			}
			images = append(images, gin.H{
				"sort":         img.Sort,
				"image_url":    img.ImageURL,
				"error":        img.ErrorMsg,
				"usage_log_id": img.UsageLogID,
			})
		}
		result = append(result, gin.H{
			"task_id":      task.TaskID,
			"status":       task.Status,
			"done_count":   doneCount,
			"total_count":  len(task.Images),
			"failed_count": failedCount,
			"cover":        cover,
			"images":       images,
			"created_at":   task.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
