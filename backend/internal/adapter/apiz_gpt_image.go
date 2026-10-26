package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/storage"
)

// apiz.ai API 配置
const (
	ApizTaskCreateURL = "/tasks/create"
	ApizTaskQueryURL  = "/tasks/query"

	apizCreateTaskAttempts   = 3
	apizCreateTaskTimeout    = 120 * time.Second
	apizCreateTaskRetryDelay = 2 * time.Second
	apizQueryTaskTimeout     = 60 * time.Second
	apizQueryErrorLimit      = 6
)

func doApizCreateTaskRequest(label string, url string, apiKey string, body []byte) ([]byte, error) {
	client := &http.Client{Timeout: apizCreateTaskTimeout}
	var lastErr error

	for attempt := 1; attempt <= apizCreateTaskAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt-1) * apizCreateTaskRetryDelay)
		}

		httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("do request: %w", err)
			log.Printf("[%s] 创建任务第 %d/%d 次失败: %v", label, attempt, apizCreateTaskAttempts, lastErr)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read response: %w", readErr)
			log.Printf("[%s] 创建任务第 %d/%d 次读取响应失败: %v", label, attempt, apizCreateTaskAttempts, lastErr)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return respBody, nil
		}

		lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			log.Printf("[%s] 创建任务第 %d/%d 次返回可重试状态: %v", label, attempt, apizCreateTaskAttempts, lastErr)
			continue
		}
		return nil, lastErr
	}

	return nil, lastErr
}

func doApizQueryTaskRequest(url string, apiKey string, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create query request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: apizQueryTaskTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do query: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read query response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ApizGPTImageRequest apiz.ai GPT 图片生成请求结构
type ApizGPTImageRequest struct {
	Model  string             `json:"model"`
	Params ApizGPTImageParams `json:"params"`
}

// ApizGPTImageParams apiz.ai GPT 图片生成参数
type ApizGPTImageParams struct {
	Prompt     string         `json:"prompt"`
	ImageSize  map[string]int `json:"image_size,omitempty"` // 图片尺寸 {"width": x, "height": y}
	Resolution string         `json:"resolution,omitempty"` // 分辨率等级 1K/2K/4K
	Quality    string         `json:"quality,omitempty"`    // 生成质量 low/medium/high
	NumImages  int            `json:"num_images,omitempty"` // 生成数量
	Image      string         `json:"image,omitempty"`      // base64图片(用于生成模式)
	ImageURLs  []string       `json:"image_urls,omitempty"` // 图片URLs(用于编辑模式)
	MimeType   string         `json:"mime_type,omitempty"`  // 图片MIME类型
}

// ApizTaskCreateResponse 创建任务响应
type ApizTaskCreateResponse struct {
	Code int `json:"code"`
	Data struct {
		TaskID string `json:"task_id"`
		Price  int    `json:"price"`
	} `json:"data"`
}

// ApizTaskQueryResponse 任务查询响应
type ApizTaskQueryResponse struct {
	Code int `json:"code"`
	Data struct {
		Status   string `json:"status"`
		Progress int    `json:"progress"`
		Result   struct {
			Images []ApizImageResult `json:"images"`
		} `json:"result,omitempty"`
		Error string `json:"error,omitempty"`
	} `json:"data"`
}

// ApizImageResult apiz.ai 图片结果
type ApizImageResult struct {
	URL string `json:"url"`
}

// ApizGPTImageResult GPT 图片生成结果
type ApizGPTImageResult struct {
	ImageData string
	MimeType  string
	ImageURL  string
}
type ApizGPTImageBatchResult struct {
	Items []*ApizGPTImageResult
}

// ApizGPTImageAdapter apiz.ai GPT 图片生成适配器
type ApizGPTImageAdapter struct {
	APIKey       string
	BaseURL      string
	OutputWidth  int
	OutputHeight int
}

// NewApizGPTImageAdapter 创建 apiz.ai GPT 图片适配器
func NewApizGPTImageAdapter(apiKey string, baseURL string) *ApizGPTImageAdapter {
	return &ApizGPTImageAdapter{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}
}

// GenerateImage 生成图片(无参考图)
func (a *ApizGPTImageAdapter) GenerateImage(ctx context.Context, prompt string, aspectRatio string, resolution string) (*ApizGPTImageResult, error) {
	batch, err := a.GenerateImages(ctx, prompt, aspectRatio, resolution, 1)
	if err != nil {
		return nil, err
	}
	if batch == nil || len(batch.Items) == 0 {
		return nil, fmt.Errorf("no image result")
	}
	return batch.Items[0], nil
}
func (a *ApizGPTImageAdapter) GenerateImages(ctx context.Context, prompt string, aspectRatio string, resolution string, n int) (*ApizGPTImageBatchResult, error) {
	return a.generate(ctx, prompt, "", "", aspectRatio, resolution, n)
}

// EditImage 编辑图片(有参考图)
func (a *ApizGPTImageAdapter) EditImage(ctx context.Context, prompt string, imageData string, mimeType string, aspectRatio string, resolution string) (*ApizGPTImageResult, error) {
	refs := []RefImage{{Data: imageData, MimeType: mimeType}}
	return a.EditImages(ctx, prompt, refs, aspectRatio, resolution)
}

// EditImages 编辑图片(有多张参考图)
func (a *ApizGPTImageAdapter) EditImages(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string) (*ApizGPTImageResult, error) {
	batch, err := a.EditImagesBatch(ctx, prompt, refs, aspectRatio, resolution, 1)
	if err != nil {
		return nil, err
	}
	if batch == nil || len(batch.Items) == 0 {
		return nil, fmt.Errorf("no image result")
	}
	return batch.Items[0], nil
}
func (a *ApizGPTImageAdapter) EditImagesBatch(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string, n int) (*ApizGPTImageBatchResult, error) {
	return a.generateWithRefs(ctx, prompt, refs, aspectRatio, resolution, n)
}

// generate 统一的图片生成/编辑方法
func (a *ApizGPTImageAdapter) generate(ctx context.Context, prompt string, imageData string, mimeType string, aspectRatio string, resolution string, n int) (*ApizGPTImageBatchResult, error) {
	refs := []RefImage{}
	if imageData != "" {
		refs = append(refs, RefImage{Data: imageData, MimeType: mimeType})
	}
	return a.generateWithRefs(ctx, prompt, refs, aspectRatio, resolution, n)
}

func (a *ApizGPTImageAdapter) generateWithRefs(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string, n int) (*ApizGPTImageBatchResult, error) {
	if n <= 0 {
		n = 1
	}
	validRefs := make([]RefImage, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref.Data) == "" && strings.TrimSpace(ref.URL) == "" {
			continue
		}
		validRefs = append(validRefs, ref)
	}

	// 根据是否有参考图选择模型
	var model string
	if len(validRefs) > 0 {
		model = "openai/gpt-image-2/edit"
	} else {
		model = "openai/gpt-image-2"
	}

	// 映射尺寸
	imageSize, _ := mapApizSizeParams(aspectRatio, resolution)
	if width, height := MatchGPTImageSize("gpt-image-2", a.OutputWidth, a.OutputHeight); width > 0 {
		imageSize = map[string]int{"width": width, "height": height}
	}

	// 构建请求参数
	params := ApizGPTImageParams{
		Prompt:     prompt,
		ImageSize:  imageSize,
		Resolution: resolution,
		Quality:    "medium", // 默认质量，后续可根据需求调整
		NumImages:  n,
	}

	// 如果有参考图，上传到临时目录并使用 image_urls
	if len(validRefs) > 0 {
		imageURLs := make([]string, 0, len(validRefs))
		for _, ref := range validRefs {
			if refURL := strings.TrimSpace(ref.URL); refURL != "" {
				imageURLs = append(imageURLs, refURL)
				continue
			}
			mimeType := strings.TrimSpace(ref.MimeType)
			if mimeType == "" {
				mimeType = "image/png"
			}
			ossURL, err := storage.SaveTempRefImage(ref.Data, mimeType)
			if err != nil {
				return nil, fmt.Errorf("上传参考图失败: %w", err)
			}
			imageURLs = append(imageURLs, ossURL)
		}
		params.ImageURLs = imageURLs

		// 清理临时文件
		defer func() {
			go func() {
				time.Sleep(storage.TempRefTTL)
				// DeleteTempRefFiles 需要本地路径，OSS URL 无法删除，依赖 TTL 清理
			}()
		}()
	}

	// 创建任务
	taskID, err := a.createTask(model, params)
	if err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	// 轮询获取结果
	result, err := a.pollTaskResult(taskID)
	if err != nil {
		return nil, fmt.Errorf("获取结果失败: %w", err)
	}

	return result, nil
}

// createTask 创建 apiz.ai 任务
func (a *ApizGPTImageAdapter) createTask(model string, params ApizGPTImageParams) (string, error) {
	reqBody := ApizGPTImageRequest{
		Model:  model,
		Params: params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	log.Printf("[Apiz-GPT] 创建任务 model=%s resolution=%s quality=%s refCount=%d", model, params.Resolution, params.Quality, len(params.ImageURLs))

	respBody, err := doApizCreateTaskRequest("Apiz-GPT", a.BaseURL+ApizTaskCreateURL, a.APIKey, body)
	if err != nil {
		return "", err
	}

	log.Printf("[Apiz-GPT] 创建任务响应: %s", string(respBody))

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
func (a *ApizGPTImageAdapter) pollTaskResult(taskID string) (*ApizGPTImageBatchResult, error) {
	maxAttempts := 60
	pollInterval := 3 * time.Second
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
			log.Printf("[Apiz-GPT] query task failed taskID=%s attempt=%d/%d consecutive=%d/%d err=%v", taskID, i+1, maxAttempts, consecutiveQueryErrors, apizQueryErrorLimit, err)
			if consecutiveQueryErrors >= apizQueryErrorLimit {
				return nil, fmt.Errorf("query task failed repeatedly: %w", err)
			}
			continue
		}

		var result ApizTaskQueryResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			consecutiveQueryErrors++
			lastQueryErr = fmt.Errorf("parse query response: %w", err)
			log.Printf("[Apiz-GPT] parse query response failed taskID=%s attempt=%d/%d consecutive=%d/%d err=%v", taskID, i+1, maxAttempts, consecutiveQueryErrors, apizQueryErrorLimit, err)
			if consecutiveQueryErrors >= apizQueryErrorLimit {
				return nil, fmt.Errorf("query task failed repeatedly: %w", lastQueryErr)
			}
			continue
		}
		consecutiveQueryErrors = 0

		log.Printf("[Apiz-GPT] query task taskID=%s status=%s progress=%d", taskID, result.Data.Status, result.Data.Progress)

		switch result.Data.Status {
		case "completed":
			if len(result.Data.Result.Images) > 0 {
				items := make([]*ApizGPTImageResult, 0, len(result.Data.Result.Images))
				for _, image := range result.Data.Result.Images {
					items = append(items, &ApizGPTImageResult{ImageURL: image.URL, MimeType: "image/png"})
				}
				return &ApizGPTImageBatchResult{Items: items}, nil
			}
			return nil, fmt.Errorf("task completed without image result")
		case "failed":
			return nil, fmt.Errorf("task failed: %s", result.Data.Error)
		case "pending", "processing":
			continue
		default:
			log.Printf("[Apiz-GPT] unknown task status: %s", result.Data.Status)
		}
	}

	if lastQueryErr != nil {
		return nil, fmt.Errorf("task timeout, last query error: %w", lastQueryErr)
	}
	return nil, fmt.Errorf("task timeout")
}

// mapApizSizeParams 将 aspect ratio 和 resolution 映射为 apiz.ai 尺寸参数
func mapApizSizeParams(aspectRatio string, resolution string) (imageSize map[string]int, quality string) {
	aspectRatio = strings.TrimSpace(aspectRatio)
	resolution = strings.TrimSpace(resolution)
	is2K := strings.EqualFold(resolution, "2K")
	is4K := strings.EqualFold(resolution, "4K")

	// 固定为高质量
	quality = "medium"

	// 根据宽高比和分辨率确定尺寸
	var width, height int
	switch aspectRatio {
	case "1:1":
		if is4K {
			width, height = 4096, 4096
		} else if is2K {
			width, height = 2048, 2048
		} else {
			width, height = 1024, 1024
		}
	case "4:3":
		if is4K {
			width, height = 2880, 2160
		} else if is2K {
			width, height = 2048, 1536
		} else {
			width, height = 1536, 1152
		}
	case "3:4":
		if is4K {
			width, height = 2160, 2880
		} else if is2K {
			width, height = 1536, 2048
		} else {
			width, height = 1152, 1536
		}
	case "16:9":
		if is4K {
			width, height = 3200, 1800
		} else if is2K {
			width, height = 2048, 1152
		} else {
			width, height = 1536, 864
		}
	case "9:16":
		if is4K {
			width, height = 1800, 3200
		} else if is2K {
			width, height = 1152, 2048
		} else {
			width, height = 864, 1536
		}
	case "4:5":
		if is4K {
			width, height = 2560, 3200
		} else if is2K {
			width, height = 1664, 2080
		} else {
			width, height = 832, 1040
		}
	case "5:4":
		if is4K {
			width, height = 3200, 2560
		} else if is2K {
			width, height = 2080, 1664
		} else {
			width, height = 1040, 832
		}
	case "21:9":
		if is4K {
			width, height = 3360, 1440
		} else if is2K {
			width, height = 2928, 1200
		} else {
			width, height = 1504, 624
		}
	case "2:1", "1:2", "3:2", "2:3", "9:21", "4:1", "1:4", "8:1", "1:8":
		width, height = mapExtendedApizSize(aspectRatio, is2K, is4K)
	default:
		if is4K {
			width, height = 4096, 4096
		} else if is2K {
			width, height = 2048, 2048
		} else {
			width, height = 1024, 1024
		}
	}

	imageSize = map[string]int{"width": width, "height": height}
	return imageSize, quality
}

// IsApizURL 判断是否为 apiz.ai 的请求地址
func mapExtendedApizSize(aspectRatio string, is2K bool, is4K bool) (int, int) {
	sizes := map[string][3][2]int{
		"2:1":  {{1408, 704}, {2304, 1152}, {3200, 1600}},
		"1:2":  {{704, 1408}, {1152, 2304}, {1600, 3200}},
		"3:2":  {{1344, 896}, {2112, 1408}, {2880, 1920}},
		"2:3":  {{896, 1344}, {1408, 2112}, {1920, 2880}},
		"9:21": {{624, 1504}, {864, 2016}, {1440, 3360}},
		"4:1":  {{1600, 400}, {2560, 640}, {3840, 960}},
		"1:4":  {{400, 1600}, {640, 2560}, {960, 3840}},
		"8:1":  {{1600, 200}, {2560, 320}, {3840, 480}},
		"1:8":  {{200, 1600}, {320, 2560}, {480, 3840}},
	}
	values, ok := sizes[aspectRatio]
	if !ok {
		return 1024, 1024
	}
	if is4K {
		return values[2][0], values[2][1]
	}
	if is2K {
		return values[1][0], values[1][1]
	}
	return values[0][0], values[0][1]
}

func IsApizURL(baseURL string) bool {
	return strings.Contains(strings.ToLower(baseURL), "apiz.ai")
}
