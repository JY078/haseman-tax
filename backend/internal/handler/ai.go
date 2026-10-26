package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/config"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// isResponsesAPI returns true if the model must use /v1/responses instead of /v1/chat/completions.
//
// Rules (based on OpenAI API reference):
//  1. ResponsesOnlyModel — models that ONLY support the Responses API:
//     o1-pro, o3-pro, o3-deep-research, o4-mini-deep-research,
//     computer-use-preview, gpt-5-codex, gpt-5-pro, gpt-5.1-codex-max
//  2. Models from the gpt-5.x / o-series families that are listed in the
//     ResponsesModel enum of the Responses API (gpt-5, gpt-5.1, gpt-5.2…,
//     o1, o3, o4, codex-mini-latest).
//  3. gpt-4.1 and its variants are in the ResponsesModel list too.
//
// All other models (gpt-4o, gpt-4, gpt-3.5-turbo, etc.) use /v1/chat/completions.
func isResponsesAPI(modelName string) bool {
	lm := strings.ToLower(modelName)

	// Exact matches for special cases
	exactMatches := map[string]struct{}{
		"codex-mini-latest":               {},
		"computer-use-preview":            {},
		"computer-use-preview-2025-03-11": {},
	}
	if _, ok := exactMatches[lm]; ok {
		return true
	}

	// Prefix-based matches
	prefixes := []string{
		"gpt-5",   // gpt-5, gpt-5-mini, gpt-5-nano, gpt-5.1, gpt-5.2, gpt-5.3, gpt-5.4, gpt-5-codex, gpt-5-pro, gpt-5.1-codex-max
		"gpt-4.1", // gpt-4.1, gpt-4.1-mini, gpt-4.1-nano
		"o1-pro",  // o1-pro, o1-pro-2025-03-19
		"o3",      // o3, o3-mini, o3-pro, o3-deep-research
		"o4",      // o4-mini, o4-mini-deep-research
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lm, prefix) {
			return true
		}
	}
	return false
}

// getSettingValue retrieves a setting value from DB
func getSettingValue(key string) string {
	var setting model.Setting
	if err := database.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

// getDefaultModelByType returns the default active model name for the given type
// (e.g. "drawing", "chat", "video"). Selection order:
//  1. Active model with is_default=true and matching type
//  2. First active model with matching type
//  3. fallback string
func getDefaultModelByType(modelType string, fallback string) string {
	var aiModels []model.AiModel
	if err := database.DB.Where("is_active = ?", true).Find(&aiModels).Error; err != nil {
		return fallback
	}

	// Pass 1: prefer is_default models
	for _, m := range aiModels {
		if m.IsDefault {
			for _, t := range m.Types {
				if t == modelType {
					return m.Name
				}
			}
		}
	}

	// Pass 2: any active model of this type
	for _, m := range aiModels {
		for _, t := range m.Types {
			if t == modelType {
				log.Printf("[getDefaultModelByType] type=%s found model=%s (default=%v)", modelType, m.Name, m.IsDefault)
				return m.Name
			}
		}
	}

	log.Printf("[getDefaultModelByType] type=%s no model found, returning fallback=%s", modelType, fallback)
	return fallback
}

const imageTaskTimeout = 45 * time.Minute

func newImageTaskContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), imageTaskTimeout)
}

// incUsageCount atomically increments usage_count for both the model and the user
func incUsageCount(userID uint, modelName string) {
	database.DB.Model(&model.User{}).Where("id = ?", userID).
		UpdateColumn("usage_count", database.DB.Raw("usage_count + 1"))
	database.DB.Model(&model.AiModel{}).Where("name = ?", modelName).
		UpdateColumn("usage_count", database.DB.Raw("usage_count + 1"))
}

// getAiModel fetches an active model record
func getAiModel(modelName string) (*model.AiModel, error) {
	var aiModel model.AiModel
	if err := database.DB.Where("name = ? AND is_active = ?", modelName, true).First(&aiModel).Error; err != nil {
		return nil, err
	}
	return &aiModel, nil
}

// ChatCompletionRequest is the incoming request body for our chat endpoint
type ChatCompletionRequest struct {
	Model       string                `json:"model"`
	Messages    []adapter.ChatMessage `json:"messages" binding:"required"`
	Stream      bool                  `json:"stream"`
	MaxTokens   int                   `json:"max_tokens"`
	Temperature float64               `json:"temperature"`
}

// Chat handles POST /api/chat - dispatches to GPT chat or response adapter based on model
func Chat(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = getDefaultModelByType("chat", "gpt-4o-mini")
	}

	apiKey := getSettingValue("chat_key")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat API key not configured"})
		return
	}

	usageLog := model.UsageLog{
		UserID:    userID,
		UsageType: "chat",
		Source:    "chat",
		Model:     req.Model,
		Status:    "failed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Record request body (mask API key, keep structure)
	// RequestBody and ResponseBody no longer stored to reduce database size
	// if reqBytes, err := json.Marshal(req); err == nil {
	// 	// usageLog.RequestBody = string(reqBytes) // not stored
	// }

	// Build prompt summary for logging
	if len(req.Messages) > 0 {
		last := req.Messages[len(req.Messages)-1]
		usageLog.Prompt = last.Content
	}

	if req.Stream {
		handleChatStream(c, req, apiKey, &usageLog)
		return
	}

	handleChatSync(c, req, apiKey, userID, &usageLog)
}

func handleChatSync(c *gin.Context, req ChatCompletionRequest, apiKey string, userID uint, usageLog *model.UsageLog) {
	var result string
	var promptTokens, completionTokens, totalTokens int

	// Get base URL for adapter selection
	baseURL := getSettingValue("access_url")

	if isResponsesAPI(req.Model) {
		// Use /v1/responses adapter
		respAdapter := adapter.NewResponseAdapter(apiKey, baseURL)
		inputs := make([]adapter.ResponseInput, len(req.Messages))
		for i, m := range req.Messages {
			inputs[i] = adapter.ResponseInput{Role: m.Role, Content: m.Content}
		}
		resp, err := respAdapter.Respond(adapter.ResponseRequest{
			Model: req.Model,
			Input: inputs,
		})
		if err != nil {
			usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
			database.DB.Create(usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("AI API error: %v", err)})
			return
		}
		if len(resp.Choices) > 0 {
			result = resp.Choices[0].Message.Content
		}
		promptTokens = resp.Usage.PromptTokens
		completionTokens = resp.Usage.CompletionTokens
		totalTokens = resp.Usage.TotalTokens
	} else {
		// Use /v1/chat/completions adapter
		chatAdapter := adapter.NewChatAdapter(apiKey, baseURL)
		chatReq := adapter.ChatRequest{
			Model:       req.Model,
			Messages:    req.Messages,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
		}
		resp, err := chatAdapter.Chat(chatReq)
		if err != nil {
			usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
			database.DB.Create(usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("AI API error: %v", err)})
			return
		}
		if len(resp.Choices) > 0 {
			result = resp.Choices[0].Message.Content
		}
		promptTokens = resp.Usage.PromptTokens
		completionTokens = resp.Usage.CompletionTokens
		totalTokens = resp.Usage.TotalTokens
	}

	usageLog.Result = result
	usageLog.PromptTokens = promptTokens
	usageLog.CompletionTokens = completionTokens
	usageLog.TotalTokens = totalTokens
	usageLog.Status = "success"
	usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())

	database.DB.Create(usageLog)
	go incUsageCount(usageLog.UserID, usageLog.Model)

	c.JSON(http.StatusOK, gin.H{
		"content":           result,
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
		"total_tokens":      totalTokens,
	})
}

func handleChatStream(c *gin.Context, req ChatCompletionRequest, apiKey string, usageLog *model.UsageLog) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Get base URL for adapter selection
	baseURL := getSettingValue("access_url")

	var fullContent strings.Builder
	var finalUsage *adapter.ChatUsage

	if isResponsesAPI(req.Model) {
		respAdapter := adapter.NewResponseAdapter(apiKey, baseURL)
		inputs := make([]adapter.ResponseInput, len(req.Messages))
		for i, m := range req.Messages {
			inputs[i] = adapter.ResponseInput{Role: m.Role, Content: m.Content}
		}

		err := respAdapter.RespondStream(adapter.ResponseRequest{
			Model:  req.Model,
			Input:  inputs,
			Stream: true,
		}, nil, func(content string, done bool, usage *adapter.ResponseUsage) {
			if done {
				if usage != nil {
					finalUsage = &adapter.ChatUsage{
						PromptTokens:     usage.PromptTokens,
						CompletionTokens: usage.CompletionTokens,
						TotalTokens:      usage.TotalTokens,
					}
					usageJSON := fmt.Sprintf(`{"usage":{"prompt_tokens":%d,"completion_tokens":%d,"total_tokens":%d}}`,
						usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
					fmt.Fprintf(c.Writer, "data: %s\n\n", usageJSON)
					flusher.Flush()
				}
				fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
			if content != "" {
				fullContent.WriteString(content)
				chunk := fmt.Sprintf(`{"choices":[{"delta":{"content":%q},"index":0}]}`, content)
				fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
				flusher.Flush()
			}
		})

		if err != nil {
			fmt.Fprintf(c.Writer, "data: {\"error\":\"%v\"}\n\n", err)
			flusher.Flush()
		}
	} else {
		chatAdapter := adapter.NewChatAdapter(apiKey, baseURL)
		chatReq := adapter.ChatRequest{
			Model:       req.Model,
			Messages:    req.Messages,
			Stream:      true,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
		}

		err := chatAdapter.ChatStream(chatReq, nil, func(content string, done bool, usage *adapter.ChatUsage) {
			if done {
				finalUsage = usage
				if usage != nil {
					usageJSON := fmt.Sprintf(`{"usage":{"prompt_tokens":%d,"completion_tokens":%d,"total_tokens":%d}}`,
						usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
					fmt.Fprintf(c.Writer, "data: %s\n\n", usageJSON)
					flusher.Flush()
				}
				fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
			if content != "" {
				fullContent.WriteString(content)
				chunk := fmt.Sprintf(`{"choices":[{"delta":{"content":%q},"index":0}]}`, content)
				fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
				flusher.Flush()
			}
		})

		if err != nil {
			fmt.Fprintf(c.Writer, "data: {\"error\":\"%v\"}\n\n", err)
			flusher.Flush()
		}
	}

	// Save usage log
	usageLog.Result = fullContent.String()
	usageLog.Status = "success"
	if finalUsage != nil {
		usageLog.PromptTokens = finalUsage.PromptTokens
		usageLog.CompletionTokens = finalUsage.CompletionTokens
		usageLog.TotalTokens = finalUsage.TotalTokens
	}
	usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
	// ResponseBody no longer stored
	database.DB.Create(usageLog)
	go incUsageCount(usageLog.UserID, usageLog.Model)
}

// RefImageItem is a single reference image in the request
type RefImageItem struct {
	Data     string `json:"data"`      // base64
	MimeType string `json:"mime_type"` // e.g. "image/jpeg"
	URL      string `json:"url,omitempty"`
}

func resolveUsageReferenceImageURL(refImages []RefImageItem, fallbackData string, fallbackMimeType string) string {
	urls := resolveUsageReferenceImageURLs(refImages, fallbackData, fallbackMimeType)
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func resolveUsageReferenceImageURLs(refImages []RefImageItem, fallbackData string, fallbackMimeType string) []string {
	urls := make([]string, 0, len(refImages))
	for _, ref := range refImages {
		if url := strings.TrimSpace(ref.URL); url != "" {
			urls = append(urls, url)
			continue
		}
		if strings.TrimSpace(ref.Data) != "" {
			if url, err := storage.SavePromptImage(ref.Data, firstNonEmpty(ref.MimeType, "image/png")); err == nil {
				urls = append(urls, url)
			} else {
				log.Printf("[UsageLog] save reference image failed: %v", err)
			}
		}
	}
	if len(urls) == 0 && strings.TrimSpace(fallbackData) != "" {
		if url, err := storage.SavePromptImage(fallbackData, firstNonEmpty(fallbackMimeType, "image/png")); err == nil {
			urls = append(urls, url)
		} else {
			log.Printf("[UsageLog] save fallback reference image failed: %v", err)
		}
	}
	return urls
}

func usageReferenceImageFields(refImages []RefImageItem, fallbackData string, fallbackMimeType string) (string, string) {
	urls := resolveUsageReferenceImageURLs(refImages, fallbackData, fallbackMimeType)
	if len(urls) == 0 {
		return "", "[]"
	}
	data, err := json.Marshal(urls)
	if err != nil {
		log.Printf("[UsageLog] marshal reference image urls failed: %v", err)
		return urls[0], "[]"
	}
	return urls[0], string(data)
}

func normalizeAsyncTaskRefImages(refImages []RefImageItem) []RefImageItem {
	normalized := make([]RefImageItem, 0, len(refImages))
	for _, ref := range refImages {
		item := RefImageItem{
			MimeType: firstNonEmpty(ref.MimeType, "image/png"),
			URL:      strings.TrimSpace(ref.URL),
		}
		if item.URL == "" && strings.TrimSpace(ref.Data) != "" {
			if url, err := storage.SavePromptImage(ref.Data, item.MimeType); err == nil {
				item.URL = url
			} else {
				log.Printf("[ImageTask] save async reference image failed: %v", err)
				item.Data = ref.Data
			}
		}
		if item.URL != "" || item.Data != "" {
			normalized = append(normalized, item)
		}
	}
	return normalized
}

func loadTaskRefImages(refImages []RefImageItem, allowURLOnly bool) ([]adapter.RefImage, error) {
	refs := make([]adapter.RefImage, 0, len(refImages))
	for _, ref := range refImages {
		refURL := strings.TrimSpace(ref.URL)
		if refURL == "" && strings.TrimSpace(ref.Data) == "" {
			continue
		}
		if refURL != "" && !allowURLOnly {
			data, mimeType, err := loadRefImageData(refURL, ref.MimeType)
			if err != nil {
				return nil, err
			}
			refs = append(refs, adapter.RefImage{Data: data, MimeType: mimeType, URL: refURL})
			continue
		}
		refs = append(refs, adapter.RefImage{
			Data:     ref.Data,
			MimeType: firstNonEmpty(ref.MimeType, "image/png"),
			URL:      refURL,
		})
	}
	return refs, nil
}

func loadGenerateRequestRefImages(req ImageGenerateRequest, allowURLOnly bool) ([]adapter.RefImage, error) {
	refs, err := loadTaskRefImages(req.RefImages, allowURLOnly)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 && strings.TrimSpace(req.RefImageData) != "" {
		refs = append(refs, adapter.RefImage{
			Data:     req.RefImageData,
			MimeType: firstNonEmpty(req.RefMimeType, "image/png"),
		})
	}
	return refs, nil
}

func publicReferenceImageURL(refURL string) string {
	clean := strings.TrimSpace(refURL)
	if clean == "" || strings.HasPrefix(strings.ToLower(clean), "http://") || strings.HasPrefix(strings.ToLower(clean), "https://") {
		return clean
	}
	baseURL := strings.TrimRight(getSettingValue("server_url"), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(getSettingValue("access_url"), "/")
	}
	if baseURL == "" {
		return clean
	}
	return baseURL + "/" + strings.TrimLeft(clean, "/")
}

func loadRefImageData(refURL string, mimeHint string) (string, string, error) {
	clean := strings.TrimSpace(refURL)
	if clean == "" {
		return "", "", fmt.Errorf("empty reference image URL")
	}
	if strings.HasPrefix(strings.ToLower(clean), "data:") {
		parts := strings.SplitN(clean, ",", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid data URL reference image")
		}
		mimeType := firstNonEmpty(mimeHint, "image/png")
		if strings.HasPrefix(parts[0], "data:") {
			header := strings.TrimPrefix(parts[0], "data:")
			if idx := strings.Index(header, ";"); idx >= 0 {
				header = header[:idx]
			}
			if header != "" {
				mimeType = header
			}
		}
		return parts[1], mimeType, nil
	}
	if strings.HasPrefix(strings.ToLower(clean), "http://") || strings.HasPrefix(strings.ToLower(clean), "https://") {
		data, mimeType, err := storage.DownloadImageToMemory(clean)
		if err != nil {
			return "", "", err
		}
		return base64.StdEncoding.EncodeToString(data), firstNonEmpty(mimeType, mimeHint, "image/png"), nil
	}

	localPath := strings.TrimPrefix(clean, "/")
	if parsed, err := url.Parse(clean); err == nil && parsed.Path != "" {
		localPath = strings.TrimPrefix(filepath.ToSlash(parsed.Path), "/")
	}
	if !strings.HasPrefix(filepath.ToSlash(localPath), "uploads/") {
		return "", "", fmt.Errorf("unsupported reference image path: %s", refURL)
	}
	resolvedPath, err := resolveLocalUploadPath(localPath)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(data), firstNonEmpty(mimeHint, mimeTypeFromImagePath(localPath)), nil
}

func mimeTypeFromImagePath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/png"
	}
}

// ImageGenerateRequest is the incoming request for image generation
type ImageGenerateRequest struct {
	Prompt      string `json:"prompt" binding:"required"`
	Model       string `json:"model"`
	AspectRatio string `json:"aspect_ratio"` // e.g. "16:9", "1:1"
	Resolution  string `json:"resolution"`   // e.g. "512", "1K", "2K", "4K"
	// Multiple reference images (preferred, up to 3)
	RefImages []RefImageItem `json:"ref_images"`
	// Single image fields kept for backward compatibility
	RefImageData string `json:"ref_image_data"` // base64
	RefMimeType  string `json:"ref_mime_type"`
}

// GenerateImage handles POST /api/image/generate
func GenerateImage(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req ImageGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey := getSettingValue("draw_key")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Drawing API key not configured"})
		return
	}

	referenceImageURL, referenceImageURLs := usageReferenceImageFields(req.RefImages, req.RefImageData, req.RefMimeType)
	usageLog := model.UsageLog{
		UserID:             userID,
		UsageType:          "drawing",
		Source:             "generate",
		Model:              req.Model,
		Prompt:             req.Prompt,
		Resolution:         req.Resolution,
		ReferenceImageURL:  referenceImageURL,
		ReferenceImageURLs: referenceImageURLs,
		Status:             "failed",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if usageLog.Model == "" {
		usageLog.Model = getDefaultModelByType("drawing", "")
	}
	if usageLog.Model == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未配置默认绘图模型，请在模型管理中设置"})
		return
	}
	drawModel := usageLog.Model

	// Record request body (redact long ref image data) - no longer stored

	// 处理提示词，添加语言指示（所有语言均明确指定，避免 AI 按提示词默认语言生成）
	prompt := req.Prompt

	// Get base URL for adapter selection
	baseURL := getSettingValue("access_url")

	// ── VIDU 分支 ───────────────────────────────────────────────────────────
	if adapter.IsViduModel(drawModel) {
		viduAdapter := adapter.NewViduAdapter(apiKey, baseURL)

		// 收集参考图（优先多图列表，兼容旧版单图字段）
		viduRefs, refErr := loadGenerateRequestRefImages(req, false)
		if refErr != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusBadRequest, gin.H{"error": "参考图读取失败: " + refErr.Error()})
			return
		}

		viduResult, err := viduAdapter.GenerateImageWithRefs(
			prompt, drawModel, req.AspectRatio, req.Resolution, viduRefs,
		)
		if err != nil {
			// usageLog.ResponseBody = fmt.Sprintf(`{"error":%q}`, err.Error()) // not stored
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
			return
		}

		// 从 CDN 下载并保存到本地
		localPath, mimeType, saveErr := storage.SaveImageFromURL(viduResult.ImageURL)
		if saveErr != nil {
			log.Printf("vidu: failed to save image from URL: %v", saveErr)
			localPath = viduResult.ImageURL
			mimeType = "image/png"
		}
		usageLog.ImageURL = localPath
		usageLog.Result = "[图片已生成]"
		// usageLog.ResponseBody = viduResult.RawResponse // not stored
		usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
		log.Printf("[DEBUG] VIDU duration: created=%v, now=%v, duration=%dms", usageLog.CreatedAt, time.Now(), usageLog.Duration)
		usageLog.Status = "success"
		database.DB.Create(&usageLog)
		go incUsageCount(usageLog.UserID, usageLog.Model)

		c.JSON(http.StatusOK, gin.H{
			"image_data": "",
			"mime_type":  mimeType,
			"text":       "",
			"image_url":  localPath,
		})
		return
	}

	// ── 腾讯 AIGC 分支 ──────────────────────────────────────────────────────
	if adapter.IsTencentAIGCModel(drawModel) {
		modelName, modelVersion := adapter.ParseTencentModelName(drawModel)
		tencentAdapter := adapter.NewTencentAIGCAdapter(apiKey, baseURL)

		// 腾讯 AIGC 需要公开 URL 传递参考图，将 base64 保存为临时文件后构造 URL。
		// 临时文件 10 分钟后删除（请求完成后立即通过 goroutine 延迟清理）。
		var refURLs []string
		var tempRefPaths []string

		collectRefs := func() {
			// 优先使用多图列表
			for _, r := range req.RefImages {
				if refURL := strings.TrimSpace(r.URL); refURL != "" {
					refURLs = append(refURLs, publicReferenceImageURL(refURL))
					continue
				}
				if r.Data == "" {
					continue
				}
				localPath, saveErr := storage.SaveTempRefImage(r.Data, r.MimeType)
				if saveErr != nil {
					log.Printf("tencent aigc: save temp ref failed: %v", saveErr)
					continue
				}
				tempRefPaths = append(tempRefPaths, localPath)
			}
			// 兼容旧版单图字段
			if len(tempRefPaths) == 0 && req.RefImageData != "" {
				localPath, saveErr := storage.SaveTempRefImage(req.RefImageData, req.RefMimeType)
				if saveErr != nil {
					log.Printf("tencent aigc: save temp ref (compat) failed: %v", saveErr)
				} else {
					tempRefPaths = append(tempRefPaths, localPath)
				}
			}
		}
		collectRefs()

		// 构造公开访问 URL（优先使用 server_url，即系统后端公开域名）
		if len(tempRefPaths) > 0 {
			baseURL := strings.TrimRight(getSettingValue("server_url"), "/")
			if baseURL == "" {
				// 兜底：尝试 access_url（不推荐，两者语义不同）
				baseURL = strings.TrimRight(getSettingValue("access_url"), "/")
			}
			for _, p := range tempRefPaths {
				// localPath 格式为 /uploads/temp_refs/xxx.png
				refURLs = append(refURLs, baseURL+p)
			}
		}

		// 10 分钟后清理临时参考图
		if len(tempRefPaths) > 0 {
			go func(paths []string) {
				time.Sleep(storage.TempRefTTL)
				storage.DeleteTempRefFiles(paths)
			}(tempRefPaths)
		}

		tencentResult, err := tencentAdapter.GenerateImageWithURLRefs(
			prompt, modelName, modelVersion, req.AspectRatio, req.Resolution, refURLs,
		)
		if err != nil {
			// usageLog.ResponseBody = fmt.Sprintf(`{"error":%q}`, err.Error()) // not stored
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
			return
		}

		// 从 CDN 下载并保存到本地，保持与 Gemini 路径一致
		localPath, mimeType, saveErr := storage.SaveImageFromURL(tencentResult.ImageURL)
		if saveErr != nil {
			log.Printf("tencent aigc: failed to save image from URL: %v", saveErr)
			localPath = tencentResult.ImageURL
			mimeType = "image/png"
		}
		usageLog.ImageURL = localPath
		usageLog.Result = "[图片已生成]"
		// usageLog.ResponseBody = tencentResult.RawResponse // not stored
		usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
		usageLog.Status = "success"
		database.DB.Create(&usageLog)
		go incUsageCount(usageLog.UserID, usageLog.Model)

		c.JSON(http.StatusOK, gin.H{
			"image_data": "",
			"mime_type":  mimeType,
			"text":       "",
			"image_url":  localPath,
		})
		return
	}

	// ── apiz.ai 分支（根据模型类型选择适配器）───────────────────────────────
	if adapter.IsApizURL(baseURL) {
		if adapter.IsGPTImageModel(drawModel) {
			// apiz.ai + GPT Image 模型
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)

			imageCtx, cancel := newImageTaskContext()
			defer cancel()

			// 判断是否有参考图
			refs, refErr := loadGenerateRequestRefImages(req, false)
			if refErr != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusBadRequest, gin.H{"error": "参考图读取失败: " + refErr.Error()})
				return
			}
			var result *adapter.ApizGPTImageResult
			var err error

			if len(refs) > 0 {
				result, err = apizAdapter.EditImages(imageCtx, prompt, refs, req.AspectRatio, req.Resolution)
			} else {
				result, err = apizAdapter.GenerateImage(imageCtx, prompt, req.AspectRatio, req.Resolution)
			}

			if err != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
				return
			}

			if strings.TrimSpace(result.ImageURL) != "" {
				imageURL := strings.TrimSpace(result.ImageURL)
				var saveErr error
				if saveErr != nil {
					database.DB.Create(&usageLog)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
					return
				}
				usageLog.ImageURL = imageURL
				usageLog.Result = "[图片已生成]"
				usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
				usageLog.Status = "success"
				database.DB.Create(&usageLog)
				go incUsageCount(usageLog.UserID, usageLog.Model)
				c.JSON(http.StatusOK, gin.H{
					"id":        usageLog.ID,
					"image_url": imageURL,
					"text":      "",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Apiz GPT 图片未返回有效图片"})
			return
		} else {
			// apiz.ai + Gemini 模型
			geminiAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)

			imageCtx, cancel := newImageTaskContext()
			defer cancel()

			// 收集参考图
			refs, refErr := loadGenerateRequestRefImages(req, false)
			if refErr != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusBadRequest, gin.H{"error": "参考图读取失败: " + refErr.Error()})
				return
			}

			var result *adapter.ApizGPTImageResult
			var err error
			if len(refs) > 0 {
				result, err = geminiAdapter.GenerateImageWithRefs(imageCtx, prompt, refs, 1, req.Resolution, req.AspectRatio)
			} else {
				result, err = geminiAdapter.GenerateImage(imageCtx, prompt, 1, req.Resolution, req.AspectRatio)
			}
			if err != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
				return
			}

			if strings.TrimSpace(result.ImageURL) != "" {
				imageURL := strings.TrimSpace(result.ImageURL)
				var saveErr error
				if saveErr != nil {
					database.DB.Create(&usageLog)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
					return
				}
				usageLog.ImageURL = imageURL
				usageLog.Result = "[图片已生成]"
				usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
				usageLog.Status = "success"
				database.DB.Create(&usageLog)
				go incUsageCount(usageLog.UserID, usageLog.Model)
				c.JSON(http.StatusOK, gin.H{
					"id":        usageLog.ID,
					"image_url": imageURL,
					"text":      "",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Apiz Gemini 图片未返回有效图片"})
			return
		}
	}

	// ── GPT Image 分支 ─────────────────────────────────────────────────────
	if adapter.IsGPTImageModel(drawModel) {
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)

		refs := make([]adapter.RefImage, 0, len(req.RefImages)+1)
		for _, r := range req.RefImages {
			if r.Data != "" || strings.TrimSpace(r.URL) != "" {
				refs = append(refs, adapter.RefImage{Data: r.Data, MimeType: r.MimeType, URL: strings.TrimSpace(r.URL)})
			}
		}
		if len(refs) == 0 && req.RefImageData != "" {
			refs = append(refs, adapter.RefImage{Data: req.RefImageData, MimeType: req.RefMimeType})
		}

		imageCtx, cancel := newImageTaskContext()
		defer cancel()
		result, err := gptAdapter.EditImage(imageCtx, prompt, refs, drawModel, req.AspectRatio, req.Resolution, 1)
		if err != nil {
			// usageLog.ResponseBody = fmt.Sprintf(`{"error":%q}`, err.Error()) // not stored
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
			return
		}

		if strings.TrimSpace(result.ImageData) != "" {
			imageURL, saveErr := storage.SaveBase64Image(result.ImageData, result.MimeType)
			if saveErr != nil {
				// usageLog.ResponseBody = fmt.Sprintf(`{"error":"save failed: %v"}`, saveErr) // not stored
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
				return
			}
			usageLog.ImageURL = imageURL
			usageLog.Result = "[图片已生成]"
			// usageLog.ResponseBody = result.RawResponse // not stored
			usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
			usageLog.Status = "success"
			database.DB.Create(&usageLog)
			go incUsageCount(usageLog.UserID, usageLog.Model)
			c.JSON(http.StatusOK, gin.H{
				"id":         usageLog.ID,
				"image_data": result.ImageData,
				"mime_type":  result.MimeType,
				"text":       "",
				"image_url":  imageURL,
			})
			return
		}

		if strings.TrimSpace(result.ImageURL) != "" {
			imageURL, mimeType, saveErr := storage.SaveImageFromURL(result.ImageURL)
			if saveErr != nil {
				// usageLog.ResponseBody = fmt.Sprintf(`{"error":"save failed: %v"}`, saveErr) // not stored
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
				return
			}
			usageLog.ImageURL = imageURL
			usageLog.Result = "[图片已生成]"
			// usageLog.ResponseBody = result.RawResponse // not stored
			usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
			usageLog.Status = "success"
			database.DB.Create(&usageLog)
			go incUsageCount(usageLog.UserID, usageLog.Model)
			c.JSON(http.StatusOK, gin.H{
				"id":         usageLog.ID,
				"image_data": "",
				"mime_type":  mimeType,
				"text":       "",
				"image_url":  imageURL,
			})
			return
		}

		// usageLog.ResponseBody = `{"error":"no image data in response"}` // not stored
		database.DB.Create(&usageLog)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
		return
	}

	// ── Gemini 分支（原有逻辑）───────────────────────────────────────────────
	geminiAdapter := adapter.NewGeminiAdapter(apiKey, baseURL)
	geminiAdapter.RetryLogger = func(attempt, maxAttempts int, retryErr error) {
		database.DB.Create(&model.UsageLog{
			UserID:       usageLog.UserID,
			UsageType:    usageLog.UsageType,
			Source:       usageLog.Source,
			Model:        usageLog.Model,
			Status:       "retry",
			ResponseBody: "", // retry not stored
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
	}

	var result *adapter.GeminiImageResult
	var err error
	actualModelName := drawModel

	ctx, cancelImage := newImageTaskContext()
	defer cancelImage()
	refs := make([]adapter.RefImage, 0, len(req.RefImages)+1)
	if len(req.RefImages) > 0 {
		for _, r := range req.RefImages {
			if r.Data != "" || strings.TrimSpace(r.URL) != "" {
				refs = append(refs, adapter.RefImage{Data: r.Data, MimeType: r.MimeType, URL: strings.TrimSpace(r.URL)})
			}
		}
	}
	if len(refs) == 0 && req.RefImageData != "" {
		refs = append(refs, adapter.RefImage{Data: req.RefImageData, MimeType: req.RefMimeType})
	}
	result, actualModelName, err = generateGeminiImageWithModelFallback(ctx, "sync-generate", geminiAdapter, prompt, refs, req.AspectRatio, req.Resolution, drawModel)

	if err != nil {
		database.DB.Create(&usageLog)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
		return
	}

	imageURL, mimeType, saveErr := saveGeminiImageResult(result)
	if saveErr != nil {
		database.DB.Create(&usageLog)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
		return
	}
	usageLog.ImageURL = imageURL
	usageLog.Model = actualModelName

	usageLog.Result = result.Text
	if usageLog.Result == "" {
		usageLog.Result = "[图片已生成]"
	}
	// usageLog.ResponseBody = result.RawResponse // not stored
	usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
	usageLog.Status = "success"
	database.DB.Create(&usageLog)
	go incUsageCount(usageLog.UserID, usageLog.Model)

	c.JSON(http.StatusOK, gin.H{
		"id":         usageLog.ID,
		"image_data": result.ImageData,
		"mime_type":  mimeType,
		"text":       result.Text,
		"image_url":  imageURL,
	})
}

// GenerateImageInternal handles POST /api/internal/image/generate (service-to-service, no user auth)
func GenerateImageInternal(c *gin.Context) {
	secret := c.GetHeader("X-Internal-Secret")
	expectedSecret := config.AppConfig.Server.InternalSecret
	if expectedSecret == "" {
		expectedSecret = getSettingValue("internal_api_secret")
	}
	if secret == "" || secret != expectedSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing internal secret"})
		return
	}

	var req ImageGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey := getSettingValue("draw_key")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Drawing API key not configured"})
		return
	}

	drawModel := req.Model
	if drawModel == "" {
		drawModel = getDefaultModelByType("drawing", "")
	}
	if drawModel == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未配置默认绘图模型，请在模型管理中设置"})
		return
	}

	// Get base URL for adapter selection
	baseURL := getSettingValue("access_url")

	prompt := req.Prompt

	// ── VIDU 分支 ───────────────────────────────────────────────────────────
	if adapter.IsViduModel(drawModel) {
		viduAdapter := adapter.NewViduAdapter(apiKey, baseURL)
		var viduRefs []adapter.RefImage
		for _, r := range req.RefImages {
			if r.Data != "" {
				viduRefs = append(viduRefs, adapter.RefImage{Data: r.Data, MimeType: r.MimeType})
			}
		}
		if len(viduRefs) == 0 && req.RefImageData != "" {
			viduRefs = append(viduRefs, adapter.RefImage{Data: req.RefImageData, MimeType: req.RefMimeType})
		}
		viduResult, err := viduAdapter.GenerateImageWithRefs(prompt, drawModel, req.AspectRatio, req.Resolution, viduRefs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
			return
		}
		localPath, mimeType, _ := storage.SaveImageFromURL(viduResult.ImageURL)
		c.JSON(http.StatusOK, gin.H{
			"image_data": "",
			"mime_type":  mimeType,
			"text":       "",
			"image_url":  localPath,
		})
		return
	}

	// ── 腾讯 AIGC 分支 ──────────────────────────────────────────────────────
	if adapter.IsTencentAIGCModel(drawModel) {
		modelName, modelVersion := adapter.ParseTencentModelName(drawModel)
		tencentAdapter := adapter.NewTencentAIGCAdapter(apiKey, baseURL)
		var refURLs []string
		var tempRefPaths []string
		for _, r := range req.RefImages {
			if refURL := strings.TrimSpace(r.URL); refURL != "" {
				refURLs = append(refURLs, publicReferenceImageURL(refURL))
				continue
			}
			if r.Data == "" {
				continue
			}
			localPath, saveErr := storage.SaveTempRefImage(r.Data, r.MimeType)
			if saveErr == nil {
				tempRefPaths = append(tempRefPaths, localPath)
			}
		}
		if len(tempRefPaths) == 0 && req.RefImageData != "" {
			localPath, saveErr := storage.SaveTempRefImage(req.RefImageData, req.RefMimeType)
			if saveErr == nil {
				tempRefPaths = append(tempRefPaths, localPath)
			}
		}
		if len(tempRefPaths) > 0 {
			baseURL := strings.TrimRight(getSettingValue("server_url"), "/")
			if baseURL == "" {
				baseURL = strings.TrimRight(getSettingValue("access_url"), "/")
			}
			for _, p := range tempRefPaths {
				refURLs = append(refURLs, baseURL+p)
			}
		}
		if len(tempRefPaths) > 0 {
			go func(paths []string) {
				time.Sleep(storage.TempRefTTL)
				storage.DeleteTempRefFiles(paths)
			}(tempRefPaths)
		}
		tencentResult, err := tencentAdapter.GenerateImageWithURLRefs(prompt, modelName, modelVersion, req.AspectRatio, req.Resolution, refURLs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
			return
		}
		localPath, mimeType, _ := storage.SaveImageFromURL(tencentResult.ImageURL)
		c.JSON(http.StatusOK, gin.H{
			"image_data": "",
			"mime_type":  mimeType,
			"text":       "",
			"image_url":  localPath,
		})
		return
	}

	// ── GPT Image 分支 ──────────────────────────────────────────────────────
	if adapter.IsGPTImageModel(drawModel) {
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)
		refs := make([]adapter.RefImage, 0, len(req.RefImages)+1)
		for _, r := range req.RefImages {
			if r.Data != "" || strings.TrimSpace(r.URL) != "" {
				refs = append(refs, adapter.RefImage{Data: r.Data, MimeType: r.MimeType, URL: strings.TrimSpace(r.URL)})
			}
		}
		if len(refs) == 0 && req.RefImageData != "" {
			refs = append(refs, adapter.RefImage{Data: req.RefImageData, MimeType: req.RefMimeType})
		}
		imageCtx, cancel := newImageTaskContext()
		defer cancel()
		result, err := gptAdapter.EditImage(imageCtx, prompt, refs, drawModel, req.AspectRatio, req.Resolution, 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
			return
		}
		if strings.TrimSpace(result.ImageData) != "" {
			imageURL, _ := storage.SaveBase64Image(result.ImageData, result.MimeType)
			c.JSON(http.StatusOK, gin.H{
				"id":         0,
				"image_data": result.ImageData,
				"mime_type":  result.MimeType,
				"text":       "",
				"image_url":  imageURL,
			})
			return
		}
		if strings.TrimSpace(result.ImageURL) != "" {
			imageURL, mimeType, _ := storage.SaveImageFromURL(result.ImageURL)
			c.JSON(http.StatusOK, gin.H{
				"id":         0,
				"image_data": "",
				"mime_type":  mimeType,
				"text":       "",
				"image_url":  imageURL,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
		return
	}

	// ── Wan 分支 ────────────────────────────────────────────────────────────
	if adapter.IsWanModel(drawModel) {
		wanAdapter := adapter.NewWanAdapter(apiKey, baseURL)
		wanResult, wanErr := wanAdapter.GenerateImage(c.Request.Context(), prompt, drawModel, req.AspectRatio, req.Resolution, 1, false, false, nil)
		if wanErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", wanErr)})
			return
		}
		if strings.TrimSpace(wanResult.ImageData) != "" {
			wanImageURL, _ := storage.SaveBase64Image(wanResult.ImageData, wanResult.MimeType)
			c.JSON(http.StatusOK, gin.H{
				"id":         0,
				"image_data": wanResult.ImageData,
				"mime_type":  wanResult.MimeType,
				"text":       "",
				"image_url":  wanImageURL,
			})
			return
		}
		if strings.TrimSpace(wanResult.ImageURL) != "" {
			wanImageURL, wanMimeType, _ := storage.SaveImageFromURL(wanResult.ImageURL)
			c.JSON(http.StatusOK, gin.H{
				"id":         0,
				"image_data": "",
				"mime_type":  wanMimeType,
				"text":       "",
				"image_url":  wanImageURL,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
		return
	}

	// ── Gemini 分支 ─────────────────────────────────────────────────────────
	geminiAdapter := adapter.NewGeminiAdapter(apiKey, baseURL)
	ctx, cancelImage := newImageTaskContext()
	defer cancelImage()
	t0Image := time.Now()
	log.Printf("[GenerateImageInternal] 开始生图 model=%s", drawModel)

	var result *adapter.GeminiImageResult
	var err error
	actualModelName := drawModel

	refs := make([]adapter.RefImage, 0, len(req.RefImages)+1)
	if len(req.RefImages) > 0 {
		for _, r := range req.RefImages {
			if r.Data != "" || strings.TrimSpace(r.URL) != "" {
				refs = append(refs, adapter.RefImage{Data: r.Data, MimeType: r.MimeType, URL: strings.TrimSpace(r.URL)})
			}
		}
	}
	if len(refs) == 0 && req.RefImageData != "" {
		refs = append(refs, adapter.RefImage{Data: req.RefImageData, MimeType: req.RefMimeType})
	}
	result, actualModelName, err = generateGeminiImageWithModelFallback(ctx, "internal-generate", geminiAdapter, prompt, refs, req.AspectRatio, req.Resolution, drawModel)

	if err != nil {
		log.Printf("[GenerateImageInternal] 生图失败 model=%s 耗时=%v err=%v", drawModel, time.Since(t0Image).Round(time.Millisecond), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image generation error: %v", err)})
		return
	}
	log.Printf("[GenerateImageInternal] 生图成功 model=%s 耗时=%v", actualModelName, time.Since(t0Image).Round(time.Millisecond))

	imageURL, mimeType, saveErr := saveGeminiImageResult(result)
	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Image save error: %v", saveErr)})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         0,
		"image_data": result.ImageData,
		"mime_type":  mimeType,
		"text":       result.Text,
		"image_url":  imageURL,
	})
}

// ChatInternal handles POST /api/internal/chat - internal service-to-service chat endpoint
// No user auth required.
func ChatInternal(c *gin.Context) {
	secret := c.GetHeader("X-Internal-Secret")
	expectedSecret := config.AppConfig.Server.InternalSecret
	if expectedSecret == "" {
		expectedSecret = getSettingValue("internal_api_secret")
	}
	if secret == "" || secret != expectedSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing internal secret"})
		return
	}

	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey := getSettingValue("chat_key")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat API key not configured"})
		return
	}

	// Get base URL for adapter selection
	baseURL := getSettingValue("access_url")

	// Use default model if not specified
	if req.Model == "" {
		req.Model = getSettingValue("default_chat_model")
		if req.Model == "" {
			req.Model = "gpt-4o-mini"
		}
	}

	if isResponsesAPI(req.Model) {
		respAdapter := adapter.NewResponseAdapter(apiKey, baseURL)
		inputs := make([]adapter.ResponseInput, len(req.Messages))
		for i, m := range req.Messages {
			inputs[i] = adapter.ResponseInput{Role: m.Role, Content: m.Content}
		}
		resp, err := respAdapter.Respond(adapter.ResponseRequest{
			Model: req.Model,
			Input: inputs,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("AI API error: %v", err)})
			return
		}
		if len(resp.Choices) > 0 {
			c.JSON(http.StatusOK, gin.H{"content": resp.Choices[0].Message.Content})
		} else {
			c.JSON(http.StatusOK, gin.H{"content": ""})
		}
		return
	}

	// Use /v1/chat/completions adapter
	chatAdapter := adapter.NewChatAdapter(apiKey, baseURL)
	chatReq := adapter.ChatRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	resp, err := chatAdapter.Chat(chatReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("AI API error: %v", err)})
		return
	}
	if len(resp.Choices) > 0 {
		c.JSON(http.StatusOK, gin.H{"content": resp.Choices[0].Message.Content})
	} else {
		c.JSON(http.StatusOK, gin.H{"content": ""})
	}
}

// promptVariationsRequest is the request body for GeneratePromptVariations.
type promptVariationsRequest struct {
	BasePrompt string `json:"base_prompt" binding:"required"`
	Count      int    `json:"count" binding:"required,min=2,max=10"`
	RefImages  []struct {
		Data     string `json:"data"`
		MimeType string `json:"mime_type"`
	} `json:"ref_images"`
}

// parseVariationsJSON strips optional markdown fences and unmarshals a JSON string array.
func parseVariationsJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw[3:], "\n"); idx != -1 {
			raw = raw[3+idx+1:]
		}
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
		raw = strings.TrimSpace(raw)
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// fallbackVariations returns a slice of size n filled with the base prompt.
func fallbackVariations(base string, n int) []string {
	s := make([]string, n)
	for i := range s {
		s[i] = base
	}
	return s
}

// GeneratePromptVariations handles POST /api/image/generate-prompt-variations.
// When ref_images are supplied it uses a vision model to analyse the product style
// before generating N style-consistent prompt variations. Without images it falls
// back to a text-only chat call.
func GeneratePromptVariations(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	_ = userIDVal.(uint)

	var req promptVariationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey := getSettingValue("chat_key")
	if apiKey == "" {
		c.JSON(http.StatusOK, gin.H{"prompts": fallbackVariations(req.BasePrompt, req.Count)})
		return
	}

	// Get base URL for adapter selection
	baseURL := getSettingValue("access_url")

	chatAdapter := adapter.NewChatAdapter(apiKey, baseURL)
	chatModel := getDefaultModelByType("chat", "gpt-4o-mini")

	systemMsg := fmt.Sprintf(
		"You are a professional image prompt engineer for AI image generation. "+
			"Your task: given a base prompt (and optionally reference product images), "+
			"generate exactly %d creative prompt variations. "+
			"Rules:\n"+
			"1. Keep the same core subject/product.\n"+
			"2. Each variation must differ in at least one dimension: lighting, color palette, camera angle, background style, mood, or artistic rendering.\n"+
			"3. If reference images are provided, analyze the product's shape, material, and visual style — all variations must be consistent with that style.\n"+
			"4. Return ONLY a valid JSON array of %d strings. No explanation, no markdown fences.", req.Count, req.Count)

	userMsg := fmt.Sprintf("Base prompt: %s\n\nGenerate %d variations now.", req.BasePrompt, req.Count)

	var rawText string
	var callErr error

	if len(req.RefImages) > 0 {
		// Vision path: include reference images for style analysis
		imgs := make([]adapter.VisionImageItem, 0, len(req.RefImages))
		for _, ri := range req.RefImages {
			imgs = append(imgs, adapter.VisionImageItem{Data: ri.Data, MimeType: ri.MimeType})
		}
		rawText, callErr = chatAdapter.AnalyzeImagesForVariations(chatModel, imgs, systemMsg, userMsg)
	} else {
		// Text-only path
		resp, err := chatAdapter.Chat(adapter.ChatRequest{
			Model: chatModel,
			Messages: []adapter.ChatMessage{
				{Role: "system", Content: systemMsg},
				{Role: "user", Content: userMsg},
			},
			MaxTokens:   1200,
			Temperature: 0.85,
		})
		if err == nil && len(resp.Choices) > 0 {
			rawText = resp.Choices[0].Message.Content
		} else {
			callErr = err
		}
	}

	if callErr != nil || rawText == "" {
		c.JSON(http.StatusOK, gin.H{"prompts": fallbackVariations(req.BasePrompt, req.Count)})
		return
	}

	variations, err := parseVariationsJSON(rawText)
	if err != nil || len(variations) == 0 {
		c.JSON(http.StatusOK, gin.H{"prompts": fallbackVariations(req.BasePrompt, req.Count)})
		return
	}

	for len(variations) < req.Count {
		variations = append(variations, req.BasePrompt)
	}
	variations = variations[:req.Count]

	c.JSON(http.StatusOK, gin.H{"prompts": variations})
}

// ============================================================
// Image Generation Async Task Handlers
// ============================================================

// imageTaskCreateRequest is the request body for creating an image generation task
type imageTaskCreateRequest struct {
	OutputWidth  int            `json:"output_width"`
	OutputHeight int            `json:"output_height"`
	Prompt       string         `json:"prompt" binding:"required"`
	Model        string         `json:"model"`
	AspectRatio  string         `json:"aspect_ratio"`
	Resolution   string         `json:"resolution"`
	RefImages    []RefImageItem `json:"ref_images"`
	BatchKey     string         `json:"batch_key"`
	BatchItemID  uint           `json:"batch_item_id"`
	BatchItemIDs []uint         `json:"batch_item_ids"`
	Count        int            `json:"count"`
}

// imageTaskResponse maps ImageGenerationTask to API response
type imageGenerationAdapterResult struct {
	ImageURL  string
	ImageURLs []string
	MimeType  string
	Model     string
}
type imageTaskResponse struct {
	TaskID      string   `json:"task_id"`
	Status      string   `json:"status"`
	ImageURL    string   `json:"image_url,omitempty"`
	ImageURLs   []string `json:"image_urls,omitempty"`
	MimeType    string   `json:"mime_type,omitempty"`
	Error       string   `json:"error,omitempty"`
	CreatedAt   string   `json:"created_at"`
	CompletedAt string   `json:"completed_at,omitempty"`
}

// CreateImageGenerationTask handles POST /api/image/generate-async
// Creates a new async image generation task and returns task_id immediately
func CreateImageGenerationTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req imageTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateCustomImageSize(req.OutputWidth, req.OutputHeight); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine which model to use
	modelName := req.Model
	if modelName == "" {
		modelName = getDefaultModelByType("drawing", "")
	}
	if modelName == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未配置默认绘图模型，请在模型管理中设置"})
		return
	}

	apiKey := ""
	if !adapter.IsAgnesImageModel(modelName) {
		apiKey = getSettingValue("draw_key")
		if apiKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Drawing API key not configured"})
			return
		}
	}

	req.RefImages = normalizeAsyncTaskRefImages(req.RefImages)
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "count must be between 1 and 4"})
		return
	}
	if req.Count > 1 && !adapter.IsGPTImageModel(modelName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前模型平台不支持单次多图", "code": "IMAGE_BATCH_UNSUPPORTED"})
		return
	}
	if req.Count == 1 {
		req.BatchItemIDs = nil
	}

	// Create task record
	taskID := fmt.Sprintf("img-%s", uuid.NewString())
	aspectRatio := req.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}

	task := model.ImageGenerationTask{
		UserID:      userID,
		TaskID:      taskID,
		BatchKey:    req.BatchKey,
		BatchItemID: req.BatchItemID,
		ImageCount:  req.Count,
		Status:      "queued",
		Prompt:      req.Prompt,
		AspectRatio: aspectRatio,
		Resolution:  req.Resolution,
		Model:       modelName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	// Link async task to batch item if batch context provided
	if req.BatchKey != "" && req.BatchItemID > 0 {
		database.DB.Model(&model.ImageBatchItem{}).
			Where("id = ? AND batch_id IN (SELECT id FROM image_batches WHERE batch_key = ? AND user_id = ?)",
				req.BatchItemID, req.BatchKey, userID).
			Updates(map[string]interface{}{"async_task_id": taskID})
	}
	if req.BatchKey != "" && len(req.BatchItemIDs) > 0 {
		database.DB.Model(&model.ImageBatchItem{}).
			Where("id IN ? AND batch_id IN (SELECT id FROM image_batches WHERE batch_key = ? AND user_id = ?)",
				req.BatchItemIDs, req.BatchKey, userID).
			Updates(map[string]interface{}{"async_task_id": taskID, "status": "generating"})
	}

	// Launch async generation in goroutine
	go runImageGenerationTask(taskID, req, modelName, userID)

	c.JSON(http.StatusOK, gin.H{
		"task_id":     taskID,
		"status":      "queued",
		"image_count": req.Count,
		"created_at":  task.CreatedAt.Format(time.RFC3339),
	})
}

// GetImageGenerationTask handles GET /api/image/generate-async/:task_id
// Returns the current status of an async image generation task
func GetImageGenerationTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("task_id")

	var task model.ImageGenerationTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	resp := imageTaskResponse{
		TaskID:    task.TaskID,
		Status:    task.Status,
		ImageURL:  task.ImageURL,
		MimeType:  task.MimeType,
		CreatedAt: task.CreatedAt.Format(time.RFC3339),
	}
	if task.ImageURLsData != "" {
		_ = json.Unmarshal([]byte(task.ImageURLsData), &resp.ImageURLs)
	}

	if task.Status == "failed" {
		resp.Error = task.ErrorMessage
	}

	if task.Status == "done" || task.Status == "failed" {
		resp.CompletedAt = task.UpdatedAt.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, resp)
}

// runImageGenerationTask performs the actual image generation asynchronously
func runImageGenerationTask(taskID string, req imageTaskCreateRequest, modelName string, userID uint) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ImageTask] panic in task %s: %v", taskID, r)
			database.DB.Model(&model.ImageGenerationTask{}).Where("task_id = ?", taskID).
				Updates(map[string]interface{}{"status": "failed", "error_message": "系统错误"})
		}
	}()

	// Update status to running
	database.DB.Model(&model.ImageGenerationTask{}).Where("task_id = ?", taskID).
		Updates(map[string]interface{}{"status": "running", "updated_at": time.Now()})

	apiKey := ""
	if !adapter.IsAgnesImageModel(modelName) {
		apiKey = getSettingValue("draw_key")
	}
	if apiKey == "" && !adapter.IsAgnesImageModel(modelName) {
		updateTaskFailed(taskID, "Drawing API key not configured", userID, req)
		return
	}

	result, genErr := generateImageThroughAsyncAdapter(taskID, apiKey, modelName, req)
	if genErr == nil && req.OutputWidth > 0 {
		result, genErr = resizeCustomImageResult(result, req.OutputWidth, req.OutputHeight)
	}
	if genErr != nil {
		updateTaskFailed(taskID, fmt.Sprintf("Image generation error: %v", genErr), userID, req)
		return
	}
	imageURL := result.ImageURL
	imageURLs := result.ImageURLs
	if len(imageURLs) == 0 && imageURL != "" {
		imageURLs = []string{imageURL}
	}
	mimeType := result.MimeType
	actualModelName := result.Model
	if actualModelName == "" {
		actualModelName = modelName
	}
	// Save successful result
	imageURLsJSON, _ := json.Marshal(imageURLs)
	database.DB.Model(&model.ImageGenerationTask{}).Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":          "done",
			"image_url":       imageURL,
			"image_urls_data": string(imageURLsJSON),
			"mime_type":       mimeType,
			"model":           actualModelName,
			"updated_at":      time.Now(),
		})

	// Update batch item if linked
	if req.BatchKey != "" && req.BatchItemID > 0 {
		updateBatchItemStatus(req.BatchKey, req.BatchItemID, imageURL, mimeType, "done", "", userID)
	}
	if req.BatchKey != "" && len(req.BatchItemIDs) > 0 {
		updateBatchItemsStatus(req.BatchKey, req.BatchItemIDs, imageURLs, mimeType, "done", "", userID)
	}

	// Record usage
	// Get task creation time for duration calculation
	var taskCreatedAt time.Time
	database.DB.Model(&model.ImageGenerationTask{}).Where("task_id = ?", taskID).Select("created_at").Scan(&taskCreatedAt)
	duration := int(time.Since(taskCreatedAt).Milliseconds())

	referenceImageURL, referenceImageURLs := usageReferenceImageFields(req.RefImages, "", "")
	log.Printf("[ImageTask] task %s duration=%dms", taskID, duration)
	for _, generatedImageURL := range imageURLs {
		usageLog := model.UsageLog{
			UserID:             userID,
			UsageType:          "drawing",
			Source:             "generate",
			Model:              actualModelName,
			Prompt:             req.Prompt,
			Resolution:         req.Resolution,
			ImageURL:           generatedImageURL,
			ReferenceImageURL:  referenceImageURL,
			ReferenceImageURLs: referenceImageURLs,
			Result:             "[图片已生成]",
			Status:             "success",
			Duration:           duration,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		database.DB.Create(&usageLog)
		go incUsageCount(userID, actualModelName)
	}

	log.Printf("[ImageTask] task %s completed successfully", taskID)
}

func generateImageThroughAsyncAdapter(taskID, apiKey, modelName string, req imageTaskCreateRequest) (imageGenerationAdapterResult, error) {
	prompt := req.Prompt
	imageURL := ""
	mimeType := ""
	actualModelName := modelName
	baseURL := getSettingValue("access_url")

	log.Printf("[ImageTask] task %s using model=%s (vidu=%v, tencent=%v, gpt=%v)", taskID, modelName, adapter.IsViduModel(modelName), adapter.IsTencentAIGCModel(modelName), adapter.IsGPTImageModel(modelName))

	if adapter.IsAgnesImageModel(modelName) {
		log.Printf("[ImageTask] task %s executing Agnes Image adapter", taskID)
		agnesAdapter := adapter.NewAgnesImageAdapter()
		ctx, cancel := newImageTaskContext()
		defer cancel()

		refs, refErr := loadTaskRefImages(req.RefImages, true)
		if refErr != nil {
			return imageGenerationAdapterResult{}, refErr
		}

		var result *adapter.ApizGPTImageResult
		var err error
		if len(refs) > 0 {
			result, err = agnesAdapter.EditImages(ctx, prompt, refs, req.AspectRatio, req.Resolution)
		} else {
			result, err = agnesAdapter.GenerateImage(ctx, prompt, req.AspectRatio, req.Resolution)
		}
		if err != nil {
			return imageGenerationAdapterResult{}, err
		}

		var saveErr error
		imageURL, mimeType, saveErr = saveApizImageResult(result)
		if saveErr != nil {
			return imageGenerationAdapterResult{}, saveErr
		}
	} else if adapter.IsViduModel(modelName) {
		log.Printf("[ImageTask] task %s executing VIDU adapter", taskID)
		viduAdapter := adapter.NewViduAdapter(apiKey, baseURL)

		viduRefs, refErr := loadTaskRefImages(req.RefImages, false)
		if refErr != nil {
			return imageGenerationAdapterResult{}, refErr
		}

		viduResult, err := viduAdapter.GenerateImageWithRefs(prompt, modelName, req.AspectRatio, req.Resolution, viduRefs)
		if err != nil {
			return imageGenerationAdapterResult{}, err
		}

		localPath, mt, saveErr := storage.SaveImageFromURL(viduResult.ImageURL)
		if saveErr != nil {
			log.Printf("vidu: failed to save image from URL: %v", saveErr)
			localPath = viduResult.ImageURL
			mt = "image/png"
		}
		imageURL = localPath
		mimeType = mt
	} else if adapter.IsTencentAIGCModel(modelName) {
		log.Printf("[ImageTask] task %s executing Tencent AIGC adapter", taskID)
		modelName2, modelVersion := adapter.ParseTencentModelName(modelName)
		tencentAdapter := adapter.NewTencentAIGCAdapter(apiKey, baseURL)

		var refURLs []string
		var tempRefPaths []string
		for _, r := range req.RefImages {
			if refURL := strings.TrimSpace(r.URL); refURL != "" {
				refURLs = append(refURLs, refURL)
				continue
			}
			if r.Data == "" {
				continue
			}
			localPath, saveErr := storage.SaveTempRefImage(r.Data, r.MimeType)
			if saveErr != nil {
				log.Printf("tencent aigc: save temp ref failed: %v", saveErr)
				continue
			}
			tempRefPaths = append(tempRefPaths, localPath)
		}
		if len(tempRefPaths) > 0 {
			publicBaseURL := strings.TrimRight(getSettingValue("server_url"), "/")
			if publicBaseURL == "" {
				publicBaseURL = strings.TrimRight(getSettingValue("access_url"), "/")
			}
			for _, p := range tempRefPaths {
				refURLs = append(refURLs, publicBaseURL+p)
			}
		}
		if len(tempRefPaths) > 0 {
			go func(paths []string) {
				time.Sleep(storage.TempRefTTL)
				storage.DeleteTempRefFiles(paths)
			}(tempRefPaths)
		}

		tencentResult, err := tencentAdapter.GenerateImageWithURLRefs(prompt, modelName2, modelVersion, req.AspectRatio, req.Resolution, refURLs)
		if err != nil {
			return imageGenerationAdapterResult{}, err
		}

		localPath, mt, saveErr := storage.SaveImageFromURL(tencentResult.ImageURL)
		if saveErr != nil {
			log.Printf("tencent aigc: failed to save image from URL: %v", saveErr)
			localPath = tencentResult.ImageURL
			mt = "image/png"
		}
		imageURL = localPath
		mimeType = mt
	} else if adapter.IsApizURL(baseURL) && adapter.IsGPTImageModel(modelName) {
		log.Printf("[ImageTask] task %s executing Apiz GPT Image adapter", taskID)
		apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)
		apizAdapter.OutputWidth, apizAdapter.OutputHeight = adapter.MatchGPTImageSize(modelName, req.OutputWidth, req.OutputHeight)
		ctx, cancel := newImageTaskContext()
		defer cancel()

		refs, refErr := loadTaskRefImages(req.RefImages, true)
		if refErr != nil {
			return imageGenerationAdapterResult{}, refErr
		}
		var apizResult *adapter.ApizGPTImageBatchResult
		var apizErr error
		if len(refs) > 0 {
			apizResult, apizErr = apizAdapter.EditImagesBatch(ctx, prompt, refs, req.AspectRatio, req.Resolution, req.Count)
		} else {
			apizResult, apizErr = apizAdapter.GenerateImages(ctx, prompt, req.AspectRatio, req.Resolution, req.Count)
		}
		if apizErr != nil {
			return imageGenerationAdapterResult{}, apizErr
		}
		if len(apizResult.Items) != req.Count {
			return imageGenerationAdapterResult{}, fmt.Errorf("image count mismatch: requested %d, got %d", req.Count, len(apizResult.Items))
		}
		imageURLs := make([]string, 0, len(apizResult.Items))
		for _, item := range apizResult.Items {
			imageURLs = append(imageURLs, strings.TrimSpace(item.ImageURL))
		}
		imageURL = imageURLs[0]
		mimeType = "image/png"
		return imageGenerationAdapterResult{ImageURL: imageURL, ImageURLs: imageURLs, MimeType: mimeType, Model: actualModelName}, nil
	} else if adapter.IsApizURL(baseURL) && adapter.IsGeminiModel(modelName) {
		log.Printf("[ImageTask] task %s executing Apiz Gemini adapter", taskID)
		geminiAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)
		ctx, cancel := newImageTaskContext()
		defer cancel()

		refs, refErr := loadTaskRefImages(req.RefImages, true)
		if refErr != nil {
			return imageGenerationAdapterResult{}, refErr
		}
		var genResult *adapter.ApizGPTImageResult
		var genErr error
		if len(refs) > 0 {
			genResult, genErr = geminiAdapter.GenerateImageWithRefs(ctx, prompt, refs, 1, req.Resolution, req.AspectRatio)
		} else {
			genResult, genErr = geminiAdapter.GenerateImage(ctx, prompt, 1, req.Resolution, req.AspectRatio)
		}
		if genErr != nil {
			return imageGenerationAdapterResult{}, genErr
		}
		imageURL = strings.TrimSpace(genResult.ImageURL)
		mimeType = "image/png"
	} else if adapter.IsGPTImageModel(modelName) {
		log.Printf("[ImageTask] task %s executing GPT Image adapter", taskID)
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)
		gptAdapter.OutputWidth, gptAdapter.OutputHeight = req.OutputWidth, req.OutputHeight
		ctx, cancel := newImageTaskContext()
		defer cancel()

		gptRefs, refErr := loadTaskRefImages(req.RefImages, true)
		if refErr != nil {
			return imageGenerationAdapterResult{}, refErr
		}
		var gptResult *adapter.GPTImageBatchResult
		var gptErr error
		if len(gptRefs) > 0 {
			gptResult, gptErr = gptAdapter.EditImages(ctx, prompt, gptRefs, modelName, req.AspectRatio, req.Resolution, req.Count)
		} else {
			gptResult, gptErr = gptAdapter.GenerateImages(ctx, prompt, modelName, req.AspectRatio, req.Resolution, req.Count)
		}
		if gptErr != nil {
			return imageGenerationAdapterResult{}, gptErr
		}
		if len(gptResult.Items) != req.Count {
			return imageGenerationAdapterResult{}, fmt.Errorf("image count mismatch: requested %d, got %d", req.Count, len(gptResult.Items))
		}
		imageURLs := make([]string, 0, len(gptResult.Items))
		for _, item := range gptResult.Items {
			var localPath string
			var saveErr error
			if strings.TrimSpace(item.ImageData) != "" {
				localPath, saveErr = storage.SaveBase64Image(item.ImageData, item.MimeType)
				mimeType = item.MimeType
			} else if strings.TrimSpace(item.ImageURL) != "" {
				localPath, mimeType, saveErr = storage.SaveImageFromURL(item.ImageURL)
			} else {
				saveErr = fmt.Errorf("empty image result")
			}
			if saveErr != nil {
				return imageGenerationAdapterResult{}, saveErr
			}
			imageURLs = append(imageURLs, localPath)
		}
		imageURL = imageURLs[0]
		return imageGenerationAdapterResult{ImageURL: imageURL, ImageURLs: imageURLs, MimeType: mimeType, Model: actualModelName}, nil
	} else {
		log.Printf("[ImageTask] task %s executing Gemini adapter (default)", taskID)
		geminiAdapter := adapter.NewGeminiAdapter(apiKey, baseURL)
		ctx, cancel := newImageTaskContext()
		defer cancel()

		refs, refErr := loadTaskRefImages(req.RefImages, true)
		if refErr != nil {
			return imageGenerationAdapterResult{}, refErr
		}
		var result *adapter.GeminiImageResult
		var err error
		result, actualModelName, err = generateGeminiImageWithModelFallback(ctx, taskID, geminiAdapter, prompt, refs, req.AspectRatio, req.Resolution, modelName)
		if err != nil {
			return imageGenerationAdapterResult{}, err
		}
		var saveErr error
		imageURL, mimeType, saveErr = saveGeminiImageResult(result)
		if saveErr != nil {
			return imageGenerationAdapterResult{}, fmt.Errorf("save generated image failed: %w", saveErr)
		}
	}

	if adapter.IsApizURL(baseURL) {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			return imageGenerationAdapterResult{}, fmt.Errorf("Apiz image URL is empty")
		}
		if mimeType == "" {
			mimeType = "image/png"
		}
		log.Printf("[ImageTask] task %s using Apiz image URL directly", taskID)
	} else {
		storedImageURL, storedMimeType, storeErr := ensureStoredImageURLForUsage(imageURL)
		if storeErr != nil {
			return imageGenerationAdapterResult{}, fmt.Errorf("save generated image failed: %w", storeErr)
		}
		imageURL = storedImageURL
		if mimeType == "" {
			mimeType = storedMimeType
		}
	}

	return imageGenerationAdapterResult{ImageURL: imageURL, MimeType: mimeType, Model: actualModelName}, nil
}

func saveApizImageResult(result *adapter.ApizGPTImageResult) (string, string, error) {
	if result == nil {
		return "", "", fmt.Errorf("empty image result")
	}
	if strings.TrimSpace(result.ImageData) != "" {
		imageURL, err := storage.SaveBase64Image(result.ImageData, result.MimeType)
		if err != nil {
			return "", "", err
		}
		return imageURL, result.MimeType, nil
	}
	if strings.TrimSpace(result.ImageURL) != "" {
		mimeType := strings.TrimSpace(result.MimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		return strings.TrimSpace(result.ImageURL), mimeType, nil
	}
	return "", "", fmt.Errorf("empty image result")
}

func ensureStoredImageURLForUsage(imageURL string) (string, string, error) {
	clean := strings.TrimSpace(imageURL)
	if clean == "" {
		return "", "", fmt.Errorf("empty image URL")
	}
	if isStoredImageURL(clean) {
		return clean, "", nil
	}
	storedURL, mimeType, err := storage.SaveImageFromURL(clean)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(storedURL) == "" {
		return "", "", fmt.Errorf("stored image URL is empty")
	}
	return storedURL, mimeType, nil
}

func isStoredImageURL(imageURL string) bool {
	clean := strings.TrimSpace(imageURL)
	if clean == "" {
		return false
	}
	lower := strings.ToLower(clean)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return true
	}
	if domain := strings.TrimSpace(storage.OSSDomain); domain != "" {
		domain = strings.ToLower(strings.TrimRight(domain, "/"))
		if strings.HasPrefix(lower, domain+"/") || lower == domain {
			return true
		}
	}
	return strings.Contains(lower, ".oss-") && strings.Contains(lower, ".aliyuncs.com/")
}

// updateTaskFailed marks a task as failed.
func updateTaskFailed(taskID string, errorMsg string, userID uint, req imageTaskCreateRequest) {
	database.DB.Model(&model.ImageGenerationTask{}).Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": errorMsg,
			"updated_at":    time.Now(),
		})
	// Update batch item if linked
	if req.BatchKey != "" && req.BatchItemID > 0 {
		updateBatchItemStatus(req.BatchKey, req.BatchItemID, "", "", "error", errorMsg, userID)
	}
	if req.BatchKey != "" && len(req.BatchItemIDs) > 0 {
		updateBatchItemsStatus(req.BatchKey, req.BatchItemIDs, nil, "", "error", errorMsg, userID)
	}
	log.Printf("[ImageTask] task %s failed: %s", taskID, errorMsg)
}

func updateBatchItemsStatus(batchKey string, batchItemIDs []uint, imageURLs []string, mimeType string, status string, errorMsg string, userID uint) {
	for index, batchItemID := range batchItemIDs {
		imageURL := ""
		if index < len(imageURLs) {
			imageURL = imageURLs[index]
		}
		updateBatchItemStatus(batchKey, batchItemID, imageURL, mimeType, status, errorMsg, userID)
	}
}

// updateBatchItemStatus updates a batch item status linked to an async task
func updateBatchItemStatus(batchKey string, batchItemID uint, imageURL string, mimeType string, status string, errorMsg string, userID uint) {
	if batchKey == "" || batchItemID == 0 {
		return
	}
	var batch model.ImageBatch
	if err := database.DB.Where("batch_key = ? AND user_id = ?", batchKey, userID).First(&batch).Error; err != nil {
		log.Printf("[ImageTask] batch not found for update: %s, %d", batchKey, batchItemID)
		return
	}
	updates := map[string]interface{}{"status": status}
	if imageURL != "" {
		updates["image_url"] = imageURL
	}
	if mimeType != "" {
		updates["mime_type"] = mimeType
	}
	if errorMsg != "" {
		updates["error"] = errorMsg
	}
	database.DB.Model(&model.ImageBatchItem{}).Where("id = ? AND batch_id = ?", batchItemID, batch.ID).Updates(updates)

	// Check if batch is complete
	var remaining int64
	database.DB.Model(&model.ImageBatchItem{}).Where("batch_id = ? AND status IN (?)", batch.ID, []string{"pending", "generating"}).Count(&remaining)
	if remaining == 0 {
		database.DB.Model(&model.ImageBatch{}).Where("id = ?", batch.ID).Update("status", "done")
	}
}

// ============== Image Batch CRUD ==============

// CreateImageBatch handles POST /api/image/batches
// Creates a new batch with items
func CreateImageBatch(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req struct {
		Title           string   `json:"title"`
		Subtitle        string   `json:"subtitle"`
		Cover           string   `json:"cover"`
		ReferenceImages []string `json:"reference_images"`
		Items           []struct {
			Prompt   string `json:"prompt"`
			ImageURL string `json:"image_url"`
			MimeType string `json:"mime_type"`
			Status   string `json:"status"`
			Error    string `json:"error"`
			ImageID  *uint  `json:"image_id"`
		} `json:"items"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	batchKey := fmt.Sprintf("batch-%d-%d", userID, time.Now().UnixMilli())

	batch := model.ImageBatch{
		UserID:          userID,
		BatchKey:        batchKey,
		Title:           req.Title,
		Subtitle:        req.Subtitle,
		Cover:           req.Cover,
		ReferenceImages: model.StringSlice(req.ReferenceImages),
		TimeText:        time.Now().Format("2006/01/02 15:04"),
		Status:          "running",
		Items:           []model.ImageBatchItem{},
	}

	for _, item := range req.Items {
		batch.Items = append(batch.Items, model.ImageBatchItem{
			Prompt:        item.Prompt,
			ImageURL:      item.ImageURL,
			MimeType:      item.MimeType,
			Status:        item.Status,
			Error:         item.Error,
			ImageID:       item.ImageID,
			SavedToDesign: false,
		})
	}

	if err := database.DB.Create(&batch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建批次失败"})
		return
	}

	// Build items response with IDs
	itemsResp := make([]gin.H, len(batch.Items))
	for i, item := range batch.Items {
		itemsResp[i] = gin.H{
			"id":        item.ID,
			"prompt":    item.Prompt,
			"status":    item.Status,
			"image_url": item.ImageURL,
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":               batch.ID,
		"key":              batch.BatchKey,
		"status":           batch.Status,
		"reference_images": batch.ReferenceImages,
		"items":            itemsResp,
	})
}

// GetImageBatches handles GET /api/image/batches
// Returns all batches for the current user (recent 20)
func GetImageBatches(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var batches []model.ImageBatch
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&batches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取批次列表失败"})
		return
	}

	// Load items for each batch
	result := make([]gin.H, 0, len(batches))
	for _, batch := range batches {
		var items []model.ImageBatchItem
		database.DB.Where("batch_id = ?", batch.ID).Order("created_at ASC").Find(&items)

		timeText := batch.TimeText
		if timeText == "" {
			timeText = batch.CreatedAt.Format("2006/01/02 15:04")
		}

		result = append(result, gin.H{
			"id":               batch.ID,
			"key":              batch.BatchKey,
			"title":            batch.Title,
			"subtitle":         batch.Subtitle,
			"cover":            batch.Cover,
			"reference_images": batch.ReferenceImages,
			"time_text":        timeText,
			"status":           batch.Status,
			"total":            len(items),
			"done_count":       countItemsByStatus(items, "done"),
			"failed_count":     countItemsByStatus(items, "error"),
			"items":            items,
			"created_at":       batch.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, result)
}

// UpdateImageBatchItem handles PUT /api/image/batches/:batch_key/items/:item_id
// Updates a single item within a batch
func UpdateImageBatchItem(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	batchKey := c.Param("batch_key")
	itemID := c.Param("item_id")

	var batch model.ImageBatch
	if err := database.DB.Where("batch_key = ? AND user_id = ?", batchKey, userID).First(&batch).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "批次不存在"})
		return
	}

	var req struct {
		ImageURL      string `json:"image_url"`
		MimeType      string `json:"mime_type"`
		Status        string `json:"status"`
		Error         string `json:"error"`
		ImageID       *uint  `json:"image_id"`
		SavedToDesign *bool  `json:"saved_to_design"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	updates := map[string]interface{}{}
	if req.ImageURL != "" {
		updates["image_url"] = req.ImageURL
	}
	if req.MimeType != "" {
		updates["mime_type"] = req.MimeType
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Error != "" {
		updates["error"] = req.Error
	}
	if req.ImageID != nil {
		updates["image_id"] = req.ImageID
	}
	if req.SavedToDesign != nil {
		updates["saved_to_design"] = *req.SavedToDesign
	}

	if err := database.DB.Model(&model.ImageBatchItem{}).Where("id = ? AND batch_id = ?", itemID, batch.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	// Check if batch is complete (all items done or error)
	var remaining int64
	database.DB.Model(&model.ImageBatchItem{}).Where("batch_id = ? AND status IN (?)", batch.ID, []string{"pending", "generating"}).Count(&remaining)

	if remaining == 0 {
		database.DB.Model(&model.ImageBatch{}).Where("id = ?", batch.ID).Update("status", "done")
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// UpdateImageBatchStatus handles PUT /api/image/batches/:batch_key/status
// Updates the overall batch status
func UpdateImageBatchStatus(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	batchKey := c.Param("batch_key")

	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	var batch model.ImageBatch
	if err := database.DB.Where("batch_key = ? AND user_id = ?", batchKey, userID).First(&batch).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "批次不存在"})
		return
	}

	database.DB.Model(&model.ImageBatch{}).Where("id = ?", batch.ID).Update("status", req.Status)

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func countItemsByStatus(items []model.ImageBatchItem, status string) int {
	count := 0
	for _, item := range items {
		if item.Status == status {
			count++
		}
	}
	return count
}
