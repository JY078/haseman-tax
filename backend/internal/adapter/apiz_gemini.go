package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"advertising-image/backend/internal/storage"
)

// apiz.ai Gemini 图片生成请求参数
type ApizGeminiParams struct {
	Prompt          string   `json:"prompt"`
	NumImages       int      `json:"num_images,omitempty"`
	Resolution      string   `json:"resolution,omitempty"`
	AspectRatio     string   `json:"aspect_ratio,omitempty"`
	SafetyTolerance string   `json:"safety_tolerance,omitempty"`
	ImageURLs       []string `json:"image_urls,omitempty"` // 参考图URLs
}

// apiz.ai Gemini 图片生成请求结构
type ApizGeminiRequest struct {
	Model   string           `json:"model"`
	Params  ApizGeminiParams `json:"params"`
	Channel interface{}      `json:"channel"`
}

// ApizGeminiAdapter apiz.ai Gemini 图片生成适配器
type ApizGeminiAdapter struct {
	APIKey  string
	BaseURL string
}

// NewApizGeminiAdapter 创建 apiz.ai Gemini 适配器
func NewApizGeminiAdapter(apiKey string, baseURL string) *ApizGeminiAdapter {
	return &ApizGeminiAdapter{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}
}

func firstOptionalString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// GenerateImage 生成图片（无参考图）
func (a *ApizGeminiAdapter) GenerateImage(ctx context.Context, prompt string, numImages int, resolution string, aspectRatio ...string) (*ApizGPTImageResult, error) {
	return a.generate(ctx, prompt, nil, numImages, resolution, firstOptionalString(aspectRatio))
}

// GenerateImageWithRefs 生成图片（有参考图）
func (a *ApizGeminiAdapter) GenerateImageWithRefs(ctx context.Context, prompt string, refs []RefImage, numImages int, resolution string, aspectRatio ...string) (*ApizGPTImageResult, error) {
	return a.generate(ctx, prompt, refs, numImages, resolution, firstOptionalString(aspectRatio))
}

// generate 统一的图片生成方法
func (a *ApizGeminiAdapter) generate(ctx context.Context, prompt string, refs []RefImage, numImages int, resolution string, aspectRatio string) (*ApizGPTImageResult, error) {
	if numImages <= 0 {
		numImages = 1
	}

	model := "fal-ai/nano-banana-2"

	params := ApizGeminiParams{
		Prompt:          prompt,
		NumImages:       numImages,
		Resolution:      strings.TrimSpace(resolution),
		AspectRatio:     strings.TrimSpace(aspectRatio),
		SafetyTolerance: "6",
	}

	// 处理参考图 - SaveTempRefImage 返回 OSS URL，直接用于 image_urls
	var tempRefPaths []string // 用于清理
	if len(refs) > 0 {
		for _, ref := range refs {
			if strings.TrimSpace(ref.Data) == "" && strings.TrimSpace(ref.URL) == "" {
				continue
			}
			if refURL := strings.TrimSpace(ref.URL); refURL != "" {
				params.ImageURLs = append(params.ImageURLs, refURL)
				continue
			}
			mimeType := strings.TrimSpace(ref.MimeType)
			if mimeType == "" {
				mimeType = "image/png"
			}
			ossURL, saveErr := storage.SaveTempRefImage(ref.Data, mimeType)
			if saveErr != nil {
				return nil, fmt.Errorf("保存参考图失败: %w", saveErr)
			}
			tempRefPaths = append(tempRefPaths, ossURL)
			params.ImageURLs = append(params.ImageURLs, ossURL)
		}

		// 延迟清理临时文件（DeleteTempRefFiles 已处理 OSS URL）
		if len(tempRefPaths) > 0 {
			go func() {
				time.Sleep(storage.TempRefTTL)
				storage.DeleteTempRefFiles(tempRefPaths)
			}()
		}
	}

	req := ApizGeminiRequest{
		Model:   model,
		Params:  params,
		Channel: nil,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 创建任务
	createURL := a.BaseURL + ApizTaskCreateURL
	log.Printf("[Apiz-Gemini] 创建任务 url=%s body=%s", createURL, string(body))

	taskID, err := a.createTask(createURL, body)
	if err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	log.Printf("[Apiz-Gemini] 任务创建成功 taskID=%s", taskID)

	// 轮询任务结果
	result, err := a.pollTaskResult(taskID)
	if err != nil {
		return nil, fmt.Errorf("获取结果失败: %w", err)
	}

	return result, nil
}

// createTask 创建 apiz.ai 任务
func (a *ApizGeminiAdapter) createTask(url string, body []byte) (string, error) {
	respBody, err := doApizCreateTaskRequest("Apiz-Gemini", url, a.APIKey, body)
	if err != nil {
		return "", err
	}

	log.Printf("[Apiz-Gemini] 创建任务响应: %s", string(respBody))

	var result ApizTaskCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	// apiz.ai 返回 code=200 表示成功，不是 code=0
	if result.Code != 0 && result.Code != 200 {
		return "", fmt.Errorf("创建任务失败: code=%d", result.Code)
	}

	return result.Data.TaskID, nil
}

// pollTaskResult 轮询任务结果
func (a *ApizGeminiAdapter) pollTaskResult(taskID string) (*ApizGPTImageResult, error) {
	maxAttempts := 60
	pollInterval := 5 * time.Second
	consecutiveQueryErrors := 0
	var lastQueryErr error

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)

		payload := map[string]string{"task_id": taskID}
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal query: %w", err)
		}

		respBody, err := doApizQueryTaskRequest(a.BaseURL+ApizTaskQueryURL, a.APIKey, body)
		if err != nil {
			consecutiveQueryErrors++
			lastQueryErr = err
			log.Printf("[Apiz-Gemini] query task failed taskID=%s attempt=%d/%d consecutive=%d/%d err=%v", taskID, i+1, maxAttempts, consecutiveQueryErrors, apizQueryErrorLimit, err)
			if consecutiveQueryErrors >= apizQueryErrorLimit {
				return nil, fmt.Errorf("query task failed repeatedly: %w", err)
			}
			continue
		}

		var result ApizTaskQueryResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			consecutiveQueryErrors++
			lastQueryErr = fmt.Errorf("parse query response: %w", err)
			log.Printf("[Apiz-Gemini] parse query response failed taskID=%s attempt=%d/%d consecutive=%d/%d err=%v", taskID, i+1, maxAttempts, consecutiveQueryErrors, apizQueryErrorLimit, err)
			if consecutiveQueryErrors >= apizQueryErrorLimit {
				return nil, fmt.Errorf("query task failed repeatedly: %w", lastQueryErr)
			}
			continue
		}
		consecutiveQueryErrors = 0

		log.Printf("[Apiz-Gemini] query task taskID=%s status=%s progress=%d", taskID, result.Data.Status, result.Data.Progress)

		switch result.Data.Status {
		case "completed":
			if len(result.Data.Result.Images) > 0 {
				return &ApizGPTImageResult{
					ImageURL: result.Data.Result.Images[0].URL,
					MimeType: "image/png",
				}, nil
			}
			return nil, fmt.Errorf("task completed without image result")
		case "failed":
			return nil, fmt.Errorf("task failed: %s", result.Data.Error)
		case "pending", "processing":
			continue
		default:
			log.Printf("[Apiz-Gemini] unknown task status: %s", result.Data.Status)
		}
	}

	if lastQueryErr != nil {
		return nil, fmt.Errorf("task timeout, last query error: %w", lastQueryErr)
	}
	return nil, fmt.Errorf("task timeout")
}

// IsGeminiModel 判断是否为 Gemini 模型
func IsGeminiModel(model string) bool {
	return strings.HasPrefix(model, "fal-ai/nano-banana") ||
		strings.EqualFold(model, "gemini") ||
		strings.EqualFold(model, "nano-banana-2")
}

// resolveGeminiModel 根据模型名解析为 apiz.ai 模型标识
func resolveGeminiModel(modelName string) string {
	modelName = strings.ToLower(modelName)
	if strings.HasPrefix(modelName, "fal-ai/nano-banana") {
		return modelName
	}
	if modelName == "gemini" || modelName == "nano-banana-2" || modelName == "nano-banana" {
		return "fal-ai/nano-banana-2"
	}
	return "fal-ai/nano-banana-2"
}
