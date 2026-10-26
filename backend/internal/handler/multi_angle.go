package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnalyzeProductRequest struct {
	ImageData string `json:"image_data" binding:"required"`
	MimeType  string `json:"mime_type"`
}

func extractJSONObject(text string) string {
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return text[start : end+1]
}

func AnalyzeProductImage(c *gin.Context) {
	if _, exists := c.Get("userID"); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req AnalyzeProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis":      "",
		"angle_prompts": gin.H{},
	})
}

type MultiAngleGenerateRequest struct {
	ImageData   string   `json:"image_data" binding:"required"`
	MimeType    string   `json:"mime_type"`
	Prompt      string   `json:"prompt"`
	View        string   `json:"view"`
	Views       []string `json:"views"`
	Quality     string   `json:"quality"`
	Resolution  string   `json:"resolution"`
	AspectRatio string   `json:"aspect_ratio"`
}

type multiAngleViewPreset struct {
	Label  string
	Prompt string
}

type multiAngleResult struct {
	View      string `json:"view"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	ImageURL  string `json:"image_url,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Error     string `json:"error,omitempty"`
	Succeeded bool   `json:"succeeded"`
}

type multiAngleTaskResponse struct {
	TaskID       string             `json:"task_id"`
	Status       string             `json:"status"`
	ErrorMessage string             `json:"error_message,omitempty"`
	ImageURL     string             `json:"image_url,omitempty"`
	MimeType     string             `json:"mime_type"`
	Quality      string             `json:"quality"`
	AspectRatio  string             `json:"aspect_ratio"`
	Results      []multiAngleResult `json:"results"`
	CreatedAt    string             `json:"created_at"`
	UpdatedAt    string             `json:"updated_at"`
}

const multiAngleTaskTimeout = imageTaskTimeout + 5*time.Minute

var multiAngleViewPresets = map[string]multiAngleViewPreset{
	"front": {
		Label:  "front",
		Prompt: "目标机位：正面平视。参考图当前画面定义为正面 0 度。保持相机在参考图主体或场景中心的正前方，视平线、拍摄距离、画面范围尽量接近参考图。不要改变原图是场景图还是单品图。",
	},
	"left45": {
		Label:  "left-front 45",
		Prompt: "目标机位：画面左前方 45 度平视。左的定义：观察者面对参考图时，画面左边那一侧。相机围绕同一个主体或场景中心，沿水平圆弧移动到画面左前方约 45 度后拍摄。主体、家具、背景和道具不旋转、不换位；不是把画面水平镜像。",
	},
	"left": {
		Label:  "left side",
		Prompt: "目标机位：画面左侧 90 度平视。左的定义：观察者面对参考图时，画面左边那一侧。相机移动到主体或场景中心的左侧约 90 度，再看向同一个中心。必须生成真实左侧侧面透视；不是让产品自己转身，不是镜像，不是重新生成一个孤立产品。",
	},
	"right45": {
		Label:  "right-front 45",
		Prompt: "目标机位：画面右前方 45 度平视。右的定义：观察者面对参考图时，画面右边那一侧。相机围绕同一个主体或场景中心，沿水平圆弧移动到画面右前方约 45 度后拍摄。主体、家具、背景和道具不旋转、不换位；不是把画面水平镜像。",
	},
	"right": {
		Label:  "right side",
		Prompt: "目标机位：画面右侧 90 度平视。右的定义：观察者面对参考图时，画面右边那一侧。相机移动到主体或场景中心的右侧约 90 度，再看向同一个中心。必须生成真实右侧侧面透视；不是让产品自己转身，不是镜像，不是重新生成一个孤立产品。",
	},
	"back": {
		Label:  "back",
		Prompt: "目标机位：背面 180 度平视。以参考图当前画面为正面，相机沿水平圆弧移动到主体或场景中心的正后方，再看向同一个中心。不可见背面只能按参考图已有结构保守推断，不能发明新造型、新配件、新材质。",
	},
	"top45": {
		Label:  "top 45",
		Prompt: "目标机位：俯拍 45 度。主体和场景固定不动，相机升高到主体或场景中心上方，并向下倾斜约 45 度拍摄。展示更多顶部表面和空间布局，但物体摆放、比例、背景结构和光照方向必须延续参考图。",
	},
	"top": {
		Label:  "top",
		Prompt: "目标机位：正上方顶视。主体和场景固定不动，相机移动到主体或场景中心正上方，垂直向下拍摄。必须保持同一布局的占地轮廓、间距、比例和边界；不要重新设计、简化或重新排列主体。",
	},
	"low": {
		Label:  "low angle",
		Prompt: "目标机位：低角度仰视。主体和场景固定不动，相机降低到平视高度以下，轻微向上看向主体或场景中心。获得低机位透视，但不要夸张高度、不要拉伸形状、不要改变产品几何和场景排列。",
	},
	"closeup": {
		Label:  "closeup",
		Prompt: "目标机位：细节特写。相机靠近参考图中的主体关键细节拍摄。只放大局部细节，不改变主体设计和上下文；如果参考图是完整场景，仍需保留能识别原场景的环境线索，不要变成孤立棚拍单品图。",
	},
}

func GenerateMultiAngleImage(c *gin.Context) {
	CreateMultiAngleTask(c)
}

func CreateMultiAngleTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req MultiAngleGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.ImageData) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少图片数据"})
		return
	}

	apiKey := strings.TrimSpace(getSettingValue("draw_key"))
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "draw_key 未配置"})
		return
	}
	baseURL := strings.TrimRight(strings.TrimSpace(getSettingValue("access_url")), "/")
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "access_url 未配置"})
		return
	}
	modelName := strings.TrimSpace(getDefaultModelByType("drawing", ""))
	if modelName == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "后台未配置默认生图模型"})
		return
	}

	quality := normalizeMultiAngleQuality(firstNonEmpty(req.Quality, req.Resolution))
	aspectRatio := normalizeMultiAngleAspectRatio(req.AspectRatio)
	viewKeys := normalizeMultiAngleViews(req.Views, req.View)
	prompts := make(map[string]string, len(viewKeys))
	for _, viewKey := range viewKeys {
		_, view := resolveMultiAngleView(viewKey)
		prompts[viewKey] = buildMultiAngleEcomPrompt(view, req.Prompt)
	}

	taskID := uuid.New().String()
	refURL, err := uploadMultiAngleReference(req.ImageData, req.MimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("上传参考图失败: %v", err)})
		return
	}

	task := model.MultiAngleTask{
		UserID:            userID,
		TaskID:            taskID,
		Status:            "queued",
		Model:             modelName,
		Quality:           quality,
		AspectRatio:       aspectRatio,
		ViewKeys:          strings.Join(viewKeys, ","),
		ReferenceImageURL: refURL,
		MimeType:          firstNonEmpty(req.MimeType, "image/png"),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	for sortIndex, viewKey := range viewKeys {
		_, view := resolveMultiAngleView(viewKey)
		result := model.MultiAngleResult{
			TaskIDRef: taskID,
			View:      viewKey,
			Label:     view.Label,
			Status:    "queued",
			Prompt:    prompts[viewKey],
			MimeType:  "image/png",
			Sort:      sortIndex,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		database.DB.Create(&result)
	}

	go runMultiAngleTask(taskID, apiKey, modelName, refURL, req.MimeType, quality, aspectRatio, viewKeys, userID)

	c.JSON(http.StatusOK, gin.H{
		"task_id":      taskID,
		"status":       "queued",
		"quality":      quality,
		"aspect_ratio": aspectRatio,
	})
}

func GetMultiAngleTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("task_id")

	var task model.MultiAngleTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Results", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort asc")
	}).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	finalizeStaleMultiAngleTask(&task)
	c.JSON(http.StatusOK, mapMultiAngleTask(task))
}

func ListMultiAngleTasks(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var tasks []model.MultiAngleTask
	if err := database.DB.
		Where("user_id = ?", userID).
		Preload("Results", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort asc")
		}).
		Order("created_at desc").
		Limit(20).
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败"})
		return
	}

	items := make([]multiAngleTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		finalizeStaleMultiAngleTask(&task)
		items = append(items, mapMultiAngleTask(task))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func finalizeStaleMultiAngleTask(task *model.MultiAngleTask) {
	if task == nil {
		return
	}
	if task.Status != "queued" && task.Status != "running" {
		return
	}
	if time.Since(task.UpdatedAt) <= multiAngleTaskTimeout {
		return
	}

	now := time.Now()
	hasSuccess := false
	firstImageURL := ""
	for i := range task.Results {
		result := &task.Results[i]
		if result.Status == "success" || strings.TrimSpace(result.ImageURL) != "" {
			hasSuccess = true
			if firstImageURL == "" {
				firstImageURL = result.ImageURL
			}
			continue
		}
		if result.Status == "queued" || result.Status == "running" {
			result.Status = "failed"
			result.ErrorMessage = "任务长时间未返回结果，请重新生成"
			result.UpdatedAt = now
		}
	}

	status := "failed"
	errorMessage := "任务长时间未返回结果，已自动结束，请重新生成"
	if hasSuccess {
		status = "partial"
		errorMessage = "部分角度长时间未返回结果，已停止等待未完成角度"
	}

	database.DB.Model(&model.MultiAngleResult{}).
		Where("task_id_ref = ? AND status IN ?", task.TaskID, []string{"queued", "running"}).
		Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": errorMessage,
			"updated_at":    now,
		})
	database.DB.Model(&model.MultiAngleTask{}).Where("task_id = ?", task.TaskID).Updates(map[string]interface{}{
		"status":          status,
		"error_message":   errorMessage,
		"first_image_url": firstImageURL,
		"updated_at":      now,
	})

	task.Status = status
	task.ErrorMessage = errorMessage
	task.FirstImageURL = firstImageURL
	task.UpdatedAt = now
}

func runMultiAngleTask(taskID, apiKey, modelName, refURL, refMimeType, quality string, aspectRatio string, viewKeys []string, userID uint) {
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("系统错误: %v", r)
			log.Printf("[MultiAngleTask] panic task=%s: %v", taskID, r)
			failMultiAngleTask(taskID, errMsg)
		}
	}()

	database.DB.Model(&model.MultiAngleTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":     "running",
		"updated_at": time.Now(),
	})

	refData, detectedMimeType, refErr := fetchSceneReferenceImageData(refURL)
	if refErr != nil || strings.TrimSpace(refData) == "" {
		errMsg := fmt.Sprintf("读取参考图失败: %v", refErr)
		if refErr == nil {
			errMsg = "参考图数据为空"
		}
		failAllMultiAngleResults(taskID, errMsg)
		failMultiAngleTask(taskID, errMsg)
		return
	}
	refMimeType = firstNonEmpty(refMimeType, detectedMimeType, "image/png")

	type angleResult struct {
		viewKey  string
		imageURL string
		success  bool
	}
	resultCh := make(chan angleResult, len(viewKeys))
	var wg sync.WaitGroup

	for _, viewKey := range viewKeys {
		viewKey := viewKey
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[MultiAngleTask] angle panic task=%s view=%s: %v", taskID, viewKey, r)
					resultCh <- angleResult{viewKey: viewKey}
				}
			}()

			_, view := resolveMultiAngleView(viewKey)
			var result model.MultiAngleResult
			if err := database.DB.Where("task_id_ref = ? AND view = ?", taskID, viewKey).First(&result).Error; err != nil {
				log.Printf("[MultiAngleTask] result missing task=%s view=%s: %v", taskID, viewKey, err)
				resultCh <- angleResult{viewKey: viewKey}
				return
			}

			database.DB.Model(&model.MultiAngleResult{}).Where("id = ?", result.ID).Updates(map[string]interface{}{
				"status":     "running",
				"updated_at": time.Now(),
			})

			generated, genErr := generateMultiAngleImageWithAsyncAdapter(taskID, apiKey, modelName, refData, refMimeType, refURL, result.Prompt, quality, aspectRatio)
			if genErr != nil {
				updateMultiAngleResultFailed(result, fmt.Sprintf("生成失败: %v", genErr))
				resultCh <- angleResult{viewKey: viewKey}
				return
			}

			imageURL := generated.ImageURL
			mimeType := firstNonEmpty(generated.MimeType, "image/png")

			database.DB.Model(&model.MultiAngleResult{}).Where("id = ?", result.ID).Updates(map[string]interface{}{
				"status":     "success",
				"image_url":  imageURL,
				"mime_type":  mimeType,
				"updated_at": time.Now(),
			})
			createMultiAngleUsageLog(userID, modelName, result.Prompt, quality, aspectRatio, viewKey, view.Label, imageURL, refURL, int(time.Since(result.CreatedAt).Milliseconds()))
			resultCh <- angleResult{viewKey: viewKey, imageURL: imageURL, success: true}
		}()
	}

	wg.Wait()
	close(resultCh)

	successCount := 0
	successURLs := make(map[string]string, len(viewKeys))
	for item := range resultCh {
		if item.success {
			successCount++
			successURLs[item.viewKey] = item.imageURL
		}
	}

	if successCount == 0 {
		failMultiAngleTask(taskID, "所有角度生成失败")
		return
	}

	firstImageURL := ""
	for _, viewKey := range viewKeys {
		if imageURL := successURLs[viewKey]; imageURL != "" {
			firstImageURL = imageURL
			break
		}
	}

	status := "done"
	errorMessage := ""
	if successCount < len(viewKeys) {
		status = "partial"
		errorMessage = fmt.Sprintf("%d/%d 个角度生成成功", successCount, len(viewKeys))
	}
	database.DB.Model(&model.MultiAngleTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":          status,
		"error_message":   errorMessage,
		"first_image_url": firstImageURL,
		"updated_at":      time.Now(),
	})
}

func failAllMultiAngleResults(taskID string, errorMessage string) {
	var results []model.MultiAngleResult
	if err := database.DB.Where("task_id_ref = ?", taskID).Find(&results).Error; err != nil {
		log.Printf("[MultiAngleTask] list results failed task=%s: %v", taskID, err)
		return
	}
	for _, result := range results {
		updateMultiAngleResultFailed(result, errorMessage)
	}
}

func updateMultiAngleResultFailed(result model.MultiAngleResult, errorMessage string) {
	database.DB.Model(&model.MultiAngleResult{}).Where("id = ?", result.ID).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": errorMessage,
		"updated_at":    time.Now(),
	})
}

func createMultiAngleUsageLog(userID uint, modelName, prompt, quality, aspectRatio, viewKey, viewLabel, imageURL, referenceImageURL string, duration int) {
	usageLog := model.UsageLog{
		UserID:            userID,
		UsageType:         "drawing",
		Source:            "multi_angle",
		Model:             modelName,
		Prompt:            prompt,
		Result:            fmt.Sprintf("[%s] 多角度图片已生成", viewLabel),
		Resolution:        quality,
		ImageURL:          imageURL,
		ReferenceImageURL: strings.TrimSpace(referenceImageURL),
		RequestBody:       fmt.Sprintf(`{"view":"%s","quality":"%s","aspect_ratio":"%s"}`, viewKey, quality, aspectRatio),
		Status:            "success",
		Duration:          duration,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	database.DB.Create(&usageLog)
	go incUsageCount(userID, modelName)
}

func failMultiAngleTask(taskID string, errorMessage string) {
	database.DB.Model(&model.MultiAngleTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": errorMessage,
		"updated_at":    time.Now(),
	})
}

func generateMultiAngleImageWithAsyncAdapter(taskID, apiKey, modelName, refData, refMimeType, refURL, prompt, quality, aspectRatio string) (imageGenerationAdapterResult, error) {
	mimeType := firstNonEmpty(refMimeType, "image/png")
	if strings.TrimSpace(refData) == "" {
		return imageGenerationAdapterResult{}, fmt.Errorf("参考图数据为空")
	}

	return generateImageThroughAsyncAdapter(taskID, apiKey, modelName, imageTaskCreateRequest{
		Prompt:      prompt,
		Model:       modelName,
		AspectRatio: aspectRatio,
		Resolution:  quality,
		RefImages: []RefImageItem{
			{Data: refData, MimeType: mimeType, URL: strings.TrimSpace(refURL)},
		},
	})
}

func mapMultiAngleTask(task model.MultiAngleTask) multiAngleTaskResponse {
	results := make([]multiAngleResult, 0, len(task.Results))
	for _, item := range task.Results {
		results = append(results, multiAngleResult{
			View:      item.View,
			Label:     item.Label,
			Status:    item.Status,
			ImageURL:  item.ImageURL,
			MimeType:  firstNonEmpty(item.MimeType, "image/png"),
			Error:     item.ErrorMessage,
			Succeeded: item.Status == "success",
		})
	}
	return multiAngleTaskResponse{
		TaskID:       task.TaskID,
		Status:       task.Status,
		ErrorMessage: task.ErrorMessage,
		ImageURL:     task.FirstImageURL,
		MimeType:     firstNonEmpty(task.MimeType, "image/png"),
		Quality:      task.Quality,
		AspectRatio:  task.AspectRatio,
		Results:      results,
		CreatedAt:    task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    task.UpdatedAt.Format(time.RFC3339),
	}
}

func normalizeMultiAngleQuality(quality string) string {
	switch strings.ToUpper(strings.TrimSpace(quality)) {
	case "1K", "2K", "4K":
		return strings.ToUpper(strings.TrimSpace(quality))
	default:
		return "2K"
	}
}

func normalizeMultiAngleAspectRatio(aspectRatio string) string {
	switch strings.TrimSpace(aspectRatio) {
	case "1:1", "4:3", "3:4", "16:9", "9:16", "21:9", "3:2", "2:3", "4:5", "5:4":
		return strings.TrimSpace(aspectRatio)
	default:
		return "1:1"
	}
}

func resolveMultiAngleView(view string) (string, multiAngleViewPreset) {
	key := strings.TrimSpace(view)
	if preset, ok := multiAngleViewPresets[key]; ok {
		return key, preset
	}
	return "left45", multiAngleViewPresets["left45"]
}

func normalizeMultiAngleViews(views []string, fallback string) []string {
	seen := make(map[string]bool, len(views)+1)
	result := make([]string, 0, len(views)+1)
	for _, view := range views {
		key := strings.TrimSpace(view)
		if _, ok := multiAngleViewPresets[key]; !ok || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, key)
	}
	if len(result) == 0 {
		key, _ := resolveMultiAngleView(fallback)
		result = append(result, key)
	}
	return result
}

func buildMultiAngleEcomPrompt(view multiAngleViewPreset, extraPrompt string) string {
	parts := []string{
		"你正在执行图生图视角重拍任务，不是文生图创作，不是产品替换，不是从参考图里抽取某个物体重新做一张图。",
		"参考图是唯一视觉依据。先判断参考图类型：如果是完整场景图，输出也必须是同一个完整场景的新机位；如果是白底或棚拍单品图，输出也必须保持白底或棚拍单品风格。",
		"锁定内容：参考图里的主体、产品外观、产品数量、家具/道具、背景结构、空间关系、比例尺度、颜色、材质、纹理、标签位置、光照方向全部保持一致。",
		"只允许改变：相机位置、观看方向和由视角变化自然产生的透视遮挡。不要移动、旋转、替换、删除或重新摆放任何主体和场景元素。",
		"当前机位要求：" + view.Prompt,
		"左右方向必须按画面坐标理解：观察者正对参考图时，画面左边就是左，画面右边就是右。不要使用镜像翻转来冒充左右视角。",
		"如果目标机位需要展示参考图不可见的侧面、顶部或背面，只能根据参考图可见结构做保守延展；不能发明新造型、新配件、新开孔、新纹理、新材质或新颜色。",
		"强禁止：不要把完整场景裁成单个产品图；不要把局部物体当成唯一主体；不要换背景；不要换产品；不要改变产品比例；不要生成多张图、拼图、文字、水印、箭头、标注或说明。",
		"输出：单张真实摄影风格图片，画面清晰，透视合理，像同一摄影师在同一现场只移动相机后重新拍摄。",
	}
	if trimmed := strings.TrimSpace(extraPrompt); trimmed != "" {
		parts = append(parts, "用户补充要求："+trimmed+"。该补充要求不能覆盖以上参考图一致性规则。")
	}
	return strings.Join(parts, " ")
}

func uploadMultiAngleReference(imageData, mimeType string) (string, error) {
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "image/png"
	}
	refURL, err := storage.SaveTempRefImage(imageData, mimeType)
	if err != nil {
		return "", fmt.Errorf("上传参考图失败: %w", err)
	}
	log.Printf("[MultiAngle] reference uploaded: %s", refURL)
	return refURL, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
