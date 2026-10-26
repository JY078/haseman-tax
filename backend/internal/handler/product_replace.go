package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
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

type ProductReplaceRequest struct {
	Mode               string     `json:"mode" binding:"required"`
	AspectRatio        string     `json:"aspect_ratio"`
	Count              int        `json:"count"`
	Quality            string     `json:"quality"`
	Prompt             string     `json:"prompt"`
	WaterEffectID      int        `json:"water_effect_id"`
	ReferenceImage1    *ImageData `json:"reference_image_1"`
	ReferenceImage2URL string     `json:"reference_image_2_url"`
	OriginalImage      *ImageData `json:"original_image"`
	ProductImage       *ImageData `json:"product_image"`
}

type ImageData struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
}

// CreateProductReplaceTask POST /api/product-replace/tasks
func CreateProductReplaceTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req ProductReplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if req.Mode != "water" && req.Mode != "product" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 mode"})
		return
	}

	if req.AspectRatio == "" {
		req.AspectRatio = "1:1"
	}

	taskID := uuid.New().String()
	imageCount := 1
	if req.Count > 0 && req.Count <= 4 {
		imageCount = req.Count
	}

	task := model.ProductReplaceTask{
		UserID:      userID,
		TaskID:      taskID,
		Status:      "queued",
		Mode:        req.Mode,
		AspectRatio: req.AspectRatio,
		ImageCount:  imageCount,
	}

	// 为每个图片创建占位记录
	for i := 0; i < imageCount; i++ {
		img := model.ProductReplaceImage{
			TaskIDRef: taskID,
			Sort:      i,
			ImageURL:  "",
		}
		task.Images = append(task.Images, img)
	}

	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	// 异步执行任务
	go runProductReplaceTask(taskID, req, imageCount)

	c.JSON(http.StatusOK, gin.H{"taskId": taskID})
}

func runProductReplaceTask(taskID string, req ProductReplaceRequest, imageCount int) {
	log.Printf("[ProductReplace] 开始处理任务: %s, count: %d, mode: %s", taskID, imageCount, req.Mode)
	database.DB.Model(&model.ProductReplaceTask{}).Where("task_id = ?", taskID).Update("status", "running")

	apiKey := getSettingValue("draw_key")
	if apiKey == "" {
		errMsg := "Drawing API key not configured"
		updateTaskError(taskID, errMsg)
		return
	}

	modelName := getDefaultModelByType("drawing", "")
	if modelName == "" {
		errMsg := "未配置默认绘图模型"
		updateTaskError(taskID, errMsg)
		return
	}

	resolution := "1K"
	if req.Quality == "2K" || req.Quality == "4K" {
		resolution = req.Quality
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	// 并发生成图片
	var wg sync.WaitGroup
	results := make([]string, imageCount)
	resultMu := sync.Mutex{}

	wg.Add(imageCount)
	for i := 0; i < imageCount; i++ {
		go func(index int) {
			defer wg.Done()
			log.Printf("[ProductReplace] 正在生成第 %d/%d 张图片", index+1, imageCount)

			var imageURL string
			if req.Mode == "water" {
				imageURL, _ = generateWaterReplaceSingle(ctx, req, apiKey, modelName, resolution)
			} else {
				imageURL, _ = generateProductReplaceSingle(ctx, req, apiKey, modelName, resolution)
			}

			resultMu.Lock()
			results[index] = imageURL
			resultMu.Unlock()
		}(i)
	}
	wg.Wait()

	finishProductReplaceTask(taskID, results)
}

func finishProductReplaceTask(taskID string, results []string) {
	tx := database.DB.Begin()
	hasError := false

	// 更新图片结果
	for i, url := range results {
		update := map[string]interface{}{"image_url": url}
		if url == "" {
			hasError = true
			update["error_msg"] = "生成失败，请稍后重试"
		}
		tx.Model(&model.ProductReplaceImage{}).Where("task_id_ref = ? AND sort = ?", taskID, i).Updates(update)
	}

	// 更新任务状态
	finalStatus := "done"
	if hasError {
		finalStatus = "failed"
	}
	tx.Model(&model.ProductReplaceTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":        finalStatus,
		"error_message": "",
	})

	tx.Commit()
}

func updateTaskError(taskID string, errMsg string) {
	database.DB.Model(&model.ProductReplaceTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": errMsg,
	})
}

// fetchProductReplaceReferenceImage 获取参考图数据
func fetchProductReplaceReferenceImage(imageURL string) (string, string, error) {
	if imageURL == "" {
		return "", "", nil
	}

	// 如果是本地 URL，转换为完整路径
	parsedURL := imageURL
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		baseURL := strings.TrimRight(getSettingValue("server_url"), "/")
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8080"
		}
		parsedURL = baseURL + imageURL
	}

	resp, err := http.Get(parsedURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", nil
	}

	// 读取 mime type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	// 读取图片数据并转为 base64
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	return base64.StdEncoding.EncodeToString(data), contentType, nil
}

func getWaterEffectLabel(id int) string {
	switch id {
	case 1:
		return "柱状水"
	case 2:
		return "花洒水"
	case 3:
		return "刀锋水"
	case 4:
		return "瀑布水"
	default:
		return ""
	}
}

// generateWaterReplaceSingle 生成单张水流替换图片
func buildWaterReplacePrompt(waterEffectID int, extra string) string {
	base := "这是一个图像编辑任务，不是重新创作图片。参考图1是最终画面的唯一主体和构图基准，必须严格保留参考图1中的水龙头外观、形状、尺寸比例、安装位置、台面、水槽、背景、相机角度、透视、裁切、光照、阴影、材质、高光、清晰度和所有非水流细节。参考图2只作为水流效果参考，只提取水流的形态、粗细、连续性、喷洒方式、透明度和动态质感；不要复制参考图2中的水龙头、背景、构图、颜色风格或其他物体。请只在参考图1水龙头的真实出水口位置添加自然水流。水流必须从出水口准确流出，方向符合重力和原图空间关系，具有真实的透明或半透明质感、折射、高光和轻微运动模糊，并匹配参考图1的光照方向、曝光、清晰度和景深。如果参考图1中的开关或把手可见，可以做最小幅度调整，使其看起来处于打开状态；如果开关不可见，不要新增开关，不要重画水龙头结构。禁止改变水龙头造型、材质、颜色、角度、位置和比例；禁止移动产品；禁止改变背景和其他物体；禁止生成并排对比图、拼图、文字、水印、手、额外产品或无关物体。最终只输出一张自然真实的编辑后图片。"
	if label := getWaterEffectLabel(waterEffectID); label != "" {
		base += " 目标水流类型：" + label + "。"
	}
	if strings.TrimSpace(extra) == "" {
		return base
	}
	return base + " 用户补充要求：" + strings.TrimSpace(extra)
}

func buildProductReplacePrompt() string {
	return "参考图1是原始场景图，参考图2是需要替换进去的新产品图。请识别参考图1中需要被替换的原产品主体，并用参考图2中的产品主体替换它。严格保留参考图1的背景、构图、相机角度、透视关系、景深、光照方向、阴影、反射、材质风格和画面比例；只替换目标产品，不改变场景中的其他物体。新产品必须完整、清晰、比例合理，与原场景自然接触，阴影和高光要与环境一致。不要把参考图2的背景带入结果，不要生成并排对比图，不要添加文字、水印或多余物体。"
}

func saveProductReplaceResult(result *adapter.ApizGPTImageResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("empty image result")
	}
	if strings.TrimSpace(result.ImageData) != "" {
		return storage.SaveBase64Image(result.ImageData, result.MimeType)
	}
	if strings.TrimSpace(result.ImageURL) != "" {
		return strings.TrimSpace(result.ImageURL), nil
	}
	return "", fmt.Errorf("AI 未返回有效图片")
}

func saveProductReplaceGPTBatchResults(result *adapter.GPTImageBatchResult) ([]string, error) {
	if result == nil || len(result.Items) == 0 {
		return nil, fmt.Errorf("empty image result")
	}
	imageURLs := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if strings.TrimSpace(item.ImageData) != "" {
			imageURL, err := storage.SaveBase64Image(item.ImageData, item.MimeType)
			if err != nil {
				return nil, err
			}
			imageURLs = append(imageURLs, imageURL)
			continue
		}
		if strings.TrimSpace(item.ImageURL) != "" {
			imageURL, _, err := storage.SaveImageFromURL(item.ImageURL)
			if err != nil {
				return nil, err
			}
			imageURLs = append(imageURLs, imageURL)
		}
	}
	if len(imageURLs) == 0 {
		return nil, fmt.Errorf("empty image result")
	}
	return imageURLs, nil
}

func saveProductReplaceGeminiResult(result *adapter.GeminiImageResult) (string, error) {
	imageURL, _, err := saveGeminiImageResult(result)
	return imageURL, err
}

func generateProductReplaceWithRefs(ctx context.Context, apiKey string, modelName string, prompt string, refs []adapter.RefImage, aspectRatio string, resolution string) (string, error) {
	baseURL := resolveGPTImageBaseURL()
	if baseURL == "" {
		baseURL = getSettingValue("access_url")
	}

	if adapter.IsApizURL(baseURL) {
		if adapter.IsGPTImageModel(modelName) {
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)
			result, err := apizAdapter.EditImages(ctx, prompt, refs, aspectRatio, resolution)
			if err != nil {
				return "", err
			}
			return saveProductReplaceResult(result)
		}

		apizAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)
		result, err := apizAdapter.GenerateImageWithRefs(ctx, prompt, refs, 1, resolution, aspectRatio)
		if err != nil {
			return "", err
		}
		return saveProductReplaceResult(result)
	}

	if adapter.IsGPTImageModel(modelName) {
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)
		result, err := gptAdapter.EditImages(ctx, prompt, refs, modelName, aspectRatio, resolution, 1)
		if err != nil {
			return "", err
		}
		imageURLs, err := saveProductReplaceGPTBatchResults(result)
		if err != nil {
			return "", err
		}
		return imageURLs[0], nil
	}

	if strings.TrimSpace(modelName) == "" {
		modelName = "gemini-2.5-flash-image-preview"
	}
	geminiAdapter := adapter.NewGeminiAdapter(apiKey, baseURL)
	result, err := geminiAdapter.GenerateImageWithRefs(ctx, prompt, refs, aspectRatio, resolution, modelName)
	if err != nil {
		return "", err
	}
	return saveProductReplaceGeminiResult(result)
}

func generateWaterReplaceSingle(ctx context.Context, req ProductReplaceRequest, apiKey string, modelName string, resolution string) (string, string) {
	prompt := buildWaterReplacePrompt(req.WaterEffectID, req.Prompt)
	originalImageData := ""
	originalMimeType := "image/png"
	if req.ReferenceImage1 != nil && req.ReferenceImage1.Data != "" {
		originalImageData = req.ReferenceImage1.Data
		if req.ReferenceImage1.MimeType != "" {
			originalMimeType = req.ReferenceImage1.MimeType
		}
	}

	// 获取参考图2的数据
	waterEffectData := ""
	waterEffectMimeType := "image/png"
	if req.ReferenceImage2URL != "" {
		data, mimeType, err := fetchProductReplaceReferenceImage(req.ReferenceImage2URL)
		if err == nil && data != "" {
			waterEffectData = data
			waterEffectMimeType = mimeType
		}
	}

	// Get base URL for adapter selection
	baseURL := resolveGPTImageBaseURL()
	if baseURL == "" {
		baseURL = getSettingValue("access_url")
	}

	// Check if it's apiz.ai URL - use new ApizGPTImageAdapter
	// Otherwise use old ApizGeminiAdapter (for yunwu.ai)
	if adapter.IsApizURL(baseURL) && adapter.IsGPTImageModel(modelName) {
		// Use new ApizGPTImageAdapter for apiz.ai
		apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)

		refs := make([]adapter.RefImage, 0, 2)
		if originalImageData != "" {
			refs = append(refs, adapter.RefImage{Data: originalImageData, MimeType: originalMimeType})
		}
		if waterEffectData != "" {
			refs = append(refs, adapter.RefImage{Data: waterEffectData, MimeType: waterEffectMimeType})
		}

		result, err := apizAdapter.EditImages(ctx, prompt, refs, req.AspectRatio, resolution)
		if err != nil {
			return "", err.Error()
		}

		imageURL, err := saveProductReplaceResult(result)
		if err != nil {
			return "", err.Error()
		}
		return imageURL, ""
	}

	// Gemini 模式 (yunwu.ai)
	refs := make([]adapter.RefImage, 0, 2)
	if originalImageData != "" {
		refs = append(refs, adapter.RefImage{Data: originalImageData, MimeType: originalMimeType})
	}
	if waterEffectData != "" {
		refs = append(refs, adapter.RefImage{Data: waterEffectData, MimeType: waterEffectMimeType})
	}

	imageURL, err := generateProductReplaceWithRefs(ctx, apiKey, modelName, prompt, refs, req.AspectRatio, resolution)
	if err != nil {
		return "", err.Error()
	}
	return imageURL, ""
}

// generateProductReplaceSingle 生成单张产品替换图片
func generateProductReplaceSingle(ctx context.Context, req ProductReplaceRequest, apiKey string, modelName string, resolution string) (string, string) {
	originalImageData := ""
	originalMimeType := "image/png"
	if req.OriginalImage != nil && req.OriginalImage.Data != "" {
		originalImageData = req.OriginalImage.Data
		if req.OriginalImage.MimeType != "" {
			originalMimeType = req.OriginalImage.MimeType
		}
	}

	productImageData := ""
	productMimeType := "image/png"
	if req.ProductImage != nil && req.ProductImage.Data != "" {
		productImageData = req.ProductImage.Data
		if req.ProductImage.MimeType != "" {
			productMimeType = req.ProductImage.MimeType
		}
	}

	prompt := buildProductReplacePrompt()

	// Get base URL for adapter selection
	baseURL := resolveGPTImageBaseURL()
	if baseURL == "" {
		baseURL = getSettingValue("access_url")
	}

	// Check if it's apiz.ai URL - use new ApizGPTImageAdapter
	// Otherwise use old ApizGeminiAdapter (for yunwu.ai)
	if adapter.IsApizURL(baseURL) && adapter.IsGPTImageModel(modelName) {
		// Use new ApizGPTImageAdapter for apiz.ai
		apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)

		refs := make([]adapter.RefImage, 0, 2)
		if originalImageData != "" {
			refs = append(refs, adapter.RefImage{Data: originalImageData, MimeType: originalMimeType})
		}
		if productImageData != "" {
			refs = append(refs, adapter.RefImage{Data: productImageData, MimeType: productMimeType})
		}

		result, err := apizAdapter.EditImages(ctx, prompt, refs, req.AspectRatio, resolution)
		if err != nil {
			return "", err.Error()
		}

		imageURL, err := saveProductReplaceResult(result)
		if err != nil {
			return "", err.Error()
		}
		return imageURL, ""
	}

	// Gemini 模式 (yunwu.ai)
	refs := make([]adapter.RefImage, 0, 2)
	if originalImageData != "" {
		refs = append(refs, adapter.RefImage{Data: originalImageData, MimeType: originalMimeType})
	}
	if productImageData != "" {
		refs = append(refs, adapter.RefImage{Data: productImageData, MimeType: productMimeType})
	}

	imageURL, err := generateProductReplaceWithRefs(ctx, apiKey, modelName, prompt, refs, req.AspectRatio, resolution)
	if err != nil {
		return "", err.Error()
	}
	return imageURL, ""
}

// GetProductReplaceTask GET /api/product-replace/tasks/:id
func GetProductReplaceTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id 不能为空"})
		return
	}

	var task model.ProductReplaceTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Images").First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	// 检查是否全部完成
	images := make([]map[string]interface{}, 0)
	allDone := true
	for _, img := range task.Images {
		item := map[string]interface{}{
			"sort":      img.Sort,
			"image_url": img.ImageURL,
			"error":     img.ErrorMsg,
			"status":    "pending",
		}
		if img.ImageURL != "" {
			item["status"] = "done"
		} else if img.ErrorMsg != "" {
			item["status"] = "error"
			allDone = false
		} else if task.Status == "running" {
			allDone = false
		}
		images = append(images, item)
	}

	// 如果所有图片都已处理完，更新任务状态
	// 只有当图片记录存在且全部处理完时才更新状态为 done
	if allDone && task.Status == "running" && len(task.Images) > 0 {
		hasError := false
		for _, img := range task.Images {
			if img.ErrorMsg != "" {
				hasError = true
				break
			}
		}
		newStatus := "done"
		if hasError {
			newStatus = "failed"
		}
		database.DB.Model(&task).Update("status", newStatus)
		task.Status = newStatus
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": task.TaskID,
		"status":  task.Status,
		"mode":    task.Mode,
		"images":  images,
		"error":   task.ErrorMessage,
	})
}

// ListProductReplaceTasks GET /api/product-replace/tasks
func ListProductReplaceTasks(c *gin.Context) {
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
	offset := (page - 1) * pageSize

	var total int64
	database.DB.Model(&model.ProductReplaceTask{}).Where("user_id = ?", userID).Count(&total)

	var tasks []model.ProductReplaceTask
	if err := database.DB.Where("user_id = ?", userID).Preload("Images").Order("created_at desc").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败"})
		return
	}

	result := make([]map[string]interface{}, 0)
	for _, task := range tasks {
		doneCount := 0
		failedCount := 0
		firstImageURL := ""
		for i, img := range task.Images {
			if img.ImageURL != "" {
				doneCount++
				if firstImageURL == "" {
					firstImageURL = img.ImageURL
				}
			} else if img.ErrorMsg != "" {
				failedCount++
			}
			if i == 0 && firstImageURL == "" {
				firstImageURL = img.ImageURL
			}
		}

		taskMap := map[string]interface{}{
			"task_id":      task.TaskID,
			"status":       task.Status,
			"mode":         task.Mode,
			"done_count":   doneCount,
			"total_count":  len(task.Images),
			"failed_count": failedCount,
			"cover":        firstImageURL,
			"created_at":   task.CreatedAt.Format(time.RFC3339),
		}
		result = append(result, taskMap)
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// SaveProductReplaceTaskToDesign POST /api/product-replace/tasks/:id/save
func SaveProductReplaceTaskToDesign(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	taskID := c.Param("id")

	var task model.ProductReplaceTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Images").First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	if task.IsSavedToDesign {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已经保存过了"})
		return
	}

	// TODO: 将图片保存到设计库
	// 这里需要实现设计库的保存逻辑

	now := time.Now()
	database.DB.Model(&task).Updates(map[string]interface{}{
		"is_saved_to_design": true,
		"saved_at":           &now,
	})

	c.JSON(http.StatusOK, gin.H{"success": true})
}
