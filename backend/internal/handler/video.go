package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

const videoUsageSource = "video_generate"

type videoGenerateRequest struct {
	Prompt          string   `json:"prompt"`
	Model           string   `json:"model"`
	Resolution      string   `json:"resolution"`
	AspectRatio     string   `json:"aspect_ratio"`
	WebhookURL      string   `json:"webhook_url"`
	Image           string   `json:"image"`
	ImageEnd        string   `json:"image_end"`
	ReferenceImages []string `json:"reference_images"`
	ReferenceVideos []string `json:"reference_videos"`
	ReferenceAudios []string `json:"reference_audios"`
	Duration        int      `json:"duration"`
	GenerateAudio   bool     `json:"generate_audio"`
	CameraFixed     bool     `json:"camera_fixed"`
	Seed            int      `json:"seed"`
}

func arkVideoAPIKey() string {
	return strings.TrimSpace(getSettingValue("video_key"))
}

func arkVideoBaseURL() string {
	return strings.TrimSpace(os.Getenv("ARK_BASE_URL"))
}

func GenerateVideo(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req videoGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入视频提示词"})
		return
	}

	modelID, modelSlug, ok := arkVideoModel(req.Model)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的视频模型"})
		return
	}
	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution != "480p" && resolution != "720p" && resolution != "1080p" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的视频清晰度"})
		return
	}
	aspectRatio := strings.TrimSpace(req.AspectRatio)
	ratio, ok := arkVideoRatio(aspectRatio)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的视频比例"})
		return
	}
	duration := req.Duration
	if duration <= 0 {
		duration = 5
	}
	maxDuration := 15
	if modelSlug == "seedance-2-5" {
		maxDuration = 30
	}
	if duration < 4 || duration > maxDuration {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("视频时长必须在 4 到 %d 秒之间", maxDuration)})
		return
	}
	seed := req.Seed
	if seed == 0 {
		seed = -1
	}

	apiKey := arkVideoAPIKey()
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未配置视频 Key，请在系统设置中填写视频 Key"})
		return
	}

	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
	defer cancel()

	content := []adapter.ArkVideoContent{{Type: "text", Text: prompt}}
	if len(req.ReferenceImages) > 9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考图片最多 9 张"})
		return
	}
	if len(req.ReferenceVideos) > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考视频最多 3 个"})
		return
	}
	if len(req.ReferenceAudios) > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考音频最多 3 个"})
		return
	}
	if media := arkVideoMediaContent(req.Image, "image_url", "first_frame"); media != nil {
		content = append(content, *media)
	}
	if media := arkVideoMediaContent(req.ImageEnd, "image_url", "last_frame"); media != nil {
		content = append(content, *media)
	}
	for _, raw := range req.ReferenceImages {
		if media := arkVideoMediaContent(raw, "image_url", "reference_image"); media != nil {
			content = append(content, *media)
		}
	}
	for _, raw := range req.ReferenceVideos {
		if media := arkVideoMediaContent(raw, "video_url", "reference_video"); media != nil {
			content = append(content, *media)
		}
	}
	for _, raw := range req.ReferenceAudios {
		if media := arkVideoMediaContent(raw, "audio_url", "reference_audio"); media != nil {
			content = append(content, *media)
		}
	}
	result, err := adapter.NewArkVideoAdapter(apiKey, arkVideoBaseURL()).Create(ctx, adapter.ArkVideoRequest{
		Model:         modelID,
		Content:       content,
		CallbackURL:   strings.TrimSpace(req.WebhookURL),
		Resolution:    resolution,
		Ratio:         ratio,
		Duration:      duration,
		GenerateAudio: req.GenerateAudio,
		CameraFixed:   false,
		Seed:          int64(seed),
		Watermark:     false,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	taskID := result.ID

	storedURL := ""
	mimeType := ""
	if result.Content.VideoURL != "" {
		recordTaskID := firstVideoNonEmpty(taskID, result.Content.VideoURL)
		storedURL, mimeType, err = ensureStoredVideoUsageLog(userID, modelID, recordTaskID, result.Content.VideoURL, videoUsageLogMeta{
			Prompt:     prompt,
			Resolution: resolution,
			Duration:   int(time.Since(startedAt).Milliseconds()),
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"video_url":    storedURL,
		"url":          storedURL,
		"source_url":   result.Content.VideoURL,
		"mime_type":    mimeType,
		"task_id":      taskID,
		"status":       result.Status,
		"error":        result.Error,
		"model":        modelSlug,
		"raw_response": result.RawResponse,
	})
}

func GetVideoTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	_, modelSlug, ok := arkVideoModel(c.Param("model"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的视频模型"})
		return
	}
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频任务 ID"})
		return
	}

	apiKey := arkVideoAPIKey()
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未配置视频 Key，请在系统设置中填写视频 Key"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := adapter.NewArkVideoAdapter(apiKey, arkVideoBaseURL()).Query(ctx, taskID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	storedURL := ""
	mimeType := ""
	status := strings.ToUpper(strings.TrimSpace(result.Status))
	meta := videoUsageLogMetaFromQuery(c)
	modelName := firstVideoNonEmpty(result.Model, modelSlug)
	if result.Content.VideoURL != "" && (isVideoTaskCompleteStatus(status) || status == "" || !isVideoTaskFailedStatus(status)) {
		storedURL, mimeType, err = ensureStoredVideoUsageLog(userID, modelName, taskID, result.Content.VideoURL, meta)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	} else if isVideoTaskFailedStatus(status) {
		ensureFailedVideoUsageLog(userID, modelName, taskID, meta)
	}

	c.JSON(http.StatusOK, gin.H{
		"video_url":    storedURL,
		"url":          storedURL,
		"source_url":   result.Content.VideoURL,
		"mime_type":    mimeType,
		"task_id":      firstVideoNonEmpty(result.ID, taskID),
		"status":       result.Status,
		"error":        result.Error,
		"model":        modelSlug,
		"raw_response": result.RawResponse,
	})
}

func firstVideoNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func storeVideoResult(videoURL string) (string, string, error) {
	return storage.SaveVideoFromURL(strings.TrimSpace(videoURL))
}

type videoUsageLogMeta struct {
	Prompt     string
	Resolution string
	Duration   int
}

func videoUsageLogMetaFromQuery(c *gin.Context) videoUsageLogMeta {
	meta := videoUsageLogMeta{
		Prompt:     strings.TrimSpace(c.Query("prompt")),
		Resolution: strings.TrimSpace(c.Query("resolution")),
	}
	if elapsedMS, err := strconv.Atoi(strings.TrimSpace(c.Query("elapsed_ms"))); err == nil && elapsedMS > 0 {
		meta.Duration = elapsedMS
		return meta
	}
	if startedAt, err := strconv.ParseInt(strings.TrimSpace(c.Query("started_at")), 10, 64); err == nil && startedAt > 0 {
		if duration := int(time.Now().UnixMilli() - startedAt); duration > 0 {
			meta.Duration = duration
		}
	}
	return meta
}

func videoUsageLogMetaUpdates(meta videoUsageLogMeta) map[string]interface{} {
	updates := map[string]interface{}{}
	if meta.Prompt != "" {
		updates["prompt"] = meta.Prompt
	}
	if meta.Resolution != "" {
		updates["resolution"] = meta.Resolution
	}
	if meta.Duration > 0 {
		updates["duration"] = meta.Duration
	}
	return updates
}

func updateVideoUsageLogMeta(logID uint, meta videoUsageLogMeta) {
	updates := videoUsageLogMetaUpdates(meta)
	if len(updates) == 0 {
		return
	}
	updates["updated_at"] = time.Now()
	database.DB.Model(&model.UsageLog{}).Where("id = ?", logID).Updates(updates)
}

func isVideoTaskCompleteStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "COMPLETED", "COMPLETE", "SUCCESS", "SUCCEEDED", "FINISHED", "DONE":
		return true
	default:
		return false
	}
}

func isVideoTaskFailedStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "FAILED", "FAIL", "ERROR", "CANCELED", "CANCELLED", "EXPIRED":
		return true
	default:
		return false
	}
}

func ensureStoredVideoUsageLog(userID uint, modelEndpoint string, taskID string, sourceURL string, meta videoUsageLogMeta) (string, string, error) {
	var usageLog model.UsageLog
	err := database.DB.
		Where("user_id = ? AND source = ? AND result = ?", userID, videoUsageSource, taskID).
		Order("created_at DESC").
		First(&usageLog).Error
	if err == nil && usageLog.Status == "success" && usageLog.ImageURL != "" {
		updateVideoUsageLogMeta(usageLog.ID, meta)
		return usageLog.ImageURL, "video/mp4", nil
	}

	storedURL, mimeType, saveErr := storeVideoResult(sourceURL)
	if saveErr != nil {
		if err == nil {
			updates := videoUsageLogMetaUpdates(meta)
			updates["status"] = "failed"
			updates["updated_at"] = time.Now()
			database.DB.Model(&usageLog).Updates(updates)
		}
		return "", "", saveErr
	}

	now := time.Now()
	if err == nil {
		updates := videoUsageLogMetaUpdates(meta)
		updates["image_url"] = storedURL
		updates["status"] = "success"
		updates["updated_at"] = now
		database.DB.Model(&usageLog).Updates(updates)
	} else {
		usageLog = model.UsageLog{
			UserID:     userID,
			UsageType:  "video",
			Source:     videoUsageSource,
			Model:      modelEndpoint,
			Prompt:     meta.Prompt,
			Result:     taskID,
			ImageURL:   storedURL,
			Resolution: meta.Resolution,
			Duration:   meta.Duration,
			Status:     "success",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		database.DB.Create(&usageLog)
	}
	go incUsageCount(userID, modelEndpoint)
	return storedURL, mimeType, nil
}

func ensureFailedVideoUsageLog(userID uint, modelEndpoint string, taskID string, meta videoUsageLogMeta) {
	var usageLog model.UsageLog
	err := database.DB.
		Where("user_id = ? AND source = ? AND result = ?", userID, videoUsageSource, taskID).
		Order("created_at DESC").
		First(&usageLog).Error
	if err == nil {
		if usageLog.Status == "success" {
			return
		}
		updates := videoUsageLogMetaUpdates(meta)
		updates["status"] = "failed"
		updates["updated_at"] = time.Now()
		database.DB.Model(&usageLog).Updates(updates)
		return
	}

	now := time.Now()
	database.DB.Create(&model.UsageLog{
		UserID:     userID,
		UsageType:  "video",
		Source:     videoUsageSource,
		Model:      modelEndpoint,
		Prompt:     meta.Prompt,
		Result:     taskID,
		Resolution: meta.Resolution,
		Duration:   meta.Duration,
		Status:     "failed",
		CreatedAt:  now,
		UpdatedAt:  now,
	})
}

func arkVideoModel(model string) (string, string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(model))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	switch normalized {
	case "sed2.5", "sed-2.5", "seedance-2.5", "seedance-2-5", "seedance-2.5-pro", "seedance-2-5-pro", "doubao-seedance-2-5-260628":
		return "doubao-seedance-2-5-260628", "seedance-2-5", true
	default:
		return "", "", false
	}
}

func arkVideoImageContent(raw, role string) *adapter.ArkVideoContent {
	return arkVideoMediaContent(raw, "image_url", role)
}

func arkVideoMediaContent(raw, mediaType, role string) *adapter.ArkVideoContent {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if !strings.HasPrefix(raw, "data:") && !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = fmt.Sprintf("data:image/png;base64,%s", raw)
	}
	content := &adapter.ArkVideoContent{Type: mediaType, Role: role}
	media := &adapter.ArkVideoImageURL{URL: raw}
	switch mediaType {
	case "video_url":
		content.VideoURL = media
	case "audio_url":
		content.AudioURL = media
	default:
		content.ImageURL = media
	}
	return content
}

func arkVideoRatio(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case "film_horizontal_21_9", "21:9":
		return "21:9", true
	case "widescreen_16_9", "16:9", "":
		return "16:9", true
	case "classic_4_3", "4:3":
		return "4:3", true
	case "square_1_1", "1:1":
		return "1:1", true
	case "traditional_3_4", "3:4":
		return "3:4", true
	case "social_story_9_16", "9:16":
		return "9:16", true
	case "adaptive":
		return "adaptive", true
	default:
		return "", false
	}
}
