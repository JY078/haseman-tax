package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const tencentAIGCBaseURL = "https://yunwu.ai"

// ── 请求结构 ──────────────────────────────────────────────────────────────────

type tencentAIGCFileInfo struct {
	Type string `json:"type"`           // "Url"
	URL  string `json:"url,omitempty"`
	Text string `json:"text,omitempty"` // 图片描述（仅 GEM 2.5/3.0 支持）
}

type tencentAIGCOutputConfig struct {
	StorageMode string `json:"storage_mode,omitempty"` // "Temporary"
	AspectRatio string `json:"aspect_ratio,omitempty"` // "1:1","16:9" 等
	Resolution  string `json:"resolution,omitempty"`   // "1080P","2K","4K" 等
}

type tencentAIGCCreateReq struct {
	ModelName    string                   `json:"model_name"`
	ModelVersion string                   `json:"model_version"`
	Prompt       string                   `json:"prompt"`
	FileInfos    []tencentAIGCFileInfo    `json:"file_infos,omitempty"`
	OutputConfig *tencentAIGCOutputConfig `json:"output_config,omitempty"`
}

// ── 响应结构 ──────────────────────────────────────────────────────────────────

// tencentAIGCCreateResp 兼容两种代理返回格式
type tencentAIGCCreateResp struct {
	// 腾讯原生包装
	Response struct {
		TaskId    string `json:"TaskId"`
		RequestId string `json:"RequestId"`
	} `json:"Response"`
	// 代理简化格式
	RequestID string `json:"request_id"`
	StatusURL string `json:"status_url"`
}

type tencentAIGCQueryResp struct {
	Response struct {
		Status string `json:"Status"` // 顶层状态
		AigcImageTask struct {
			Status string `json:"Status"` // 任务级状态：FINISH / PROCESSING / FAIL
			Output struct {
				FileInfos []struct {
					FileUrl    string `json:"FileUrl"`
					ExpireTime string `json:"ExpireTime"`
				} `json:"FileInfos"`
			} `json:"Output"`
		} `json:"AigcImageTask"`
	} `json:"Response"`
}

// ── 模型辅助函数 ───────────────────────────────────────────────────────────────

// IsTencentAIGCModel 判断模型名是否应走腾讯 AIGC 接口。
// 系统中腾讯 AIGC 模型命名约定：<ModelName>-<Version>，如 GEM-2.5、Qwen-0925。
// Gemini 原生模型名为小写 gemini-xxx，不会误判。
func IsTencentAIGCModel(name string) bool {
	base := strings.SplitN(name, "-", 2)[0]
	switch strings.ToUpper(base) {
	case "GEM", "QWEN", "HUNYUAN", "JIMENG":
		return true
	}
	return false
}

// ParseTencentModelName 将 "GEM-2.5" 拆分为 ("GEM","2.5")。
// 无版本号时 version 返回空字符串。
func ParseTencentModelName(name string) (modelName, modelVersion string) {
	parts := strings.SplitN(name, "-", 2)
	modelName = parts[0]
	if len(parts) == 2 {
		modelVersion = parts[1]
	}
	return
}

// mapResolution 将系统分辨率标识转换为腾讯 AIGC 接受的格式。
func mapResolution(res string) string {
	switch res {
	case "512":
		return "720P"
	case "2K":
		return "2K"
	case "4K":
		return "4K"
	default: // "", "1K"
		return "1080P"
	}
}

// ── Adapter ────────────────────────────────────────────────────────────────────

// TencentAIGCAdapter 对接腾讯 AIGC 异步生图接口。
type TencentAIGCAdapter struct {
	APIKey  string
	BaseURL string
}

func NewTencentAIGCAdapter(apiKey string, baseURL string) *TencentAIGCAdapter {
	if baseURL == "" {
		baseURL = tencentAIGCBaseURL
	}
	return &TencentAIGCAdapter{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

// createTask 提交生图任务，返回 task_id。
func (a *TencentAIGCAdapter) createTask(req tencentAIGCCreateReq) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", a.BaseURL+"/tencent-vod/v1/aigc-image", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := shortClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var createResp tencentAIGCCreateResp
	if err := json.Unmarshal(respBody, &createResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	// 优先取腾讯原生格式，降级取代理简化格式
	taskID := createResp.Response.TaskId
	if taskID == "" {
		taskID = createResp.RequestID
	}
	if taskID == "" {
		return "", fmt.Errorf("no task_id in response: %s", string(respBody))
	}
	return taskID, nil
}

// pollTask 轮询任务直到完成或超时，返回 CDN 图片 URL。
// 采用逐步增长的间隔：2s → 3s → 4s → … 最大 10s。
func (a *TencentAIGCAdapter) pollTask(taskID string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	interval := 2 * time.Second
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		if interval < 10*time.Second {
			interval += time.Second
		}

		url := fmt.Sprintf("%s/tencent-vod/v1/query/%s", a.BaseURL, taskID)
		httpReq, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

		resp, err := pollClient.Do(httpReq)
		if err != nil {
			continue
		}
		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		var queryResp tencentAIGCQueryResp
		if err := json.Unmarshal(respBody, &queryResp); err != nil {
			continue
		}

		// 取任务级状态，降级取顶层状态
		status := queryResp.Response.AigcImageTask.Status
		if status == "" {
			status = queryResp.Response.Status
		}

		switch strings.ToUpper(status) {
		case "FINISH":
			files := queryResp.Response.AigcImageTask.Output.FileInfos
			if len(files) == 0 {
				return "", fmt.Errorf("task finished but output has no files")
			}
			return files[0].FileUrl, nil
		case "FAIL":
			return "", fmt.Errorf("tencent AIGC task failed (task_id=%s): %s", taskID, string(respBody))
		}
		// PROCESSING 或其他状态继续轮询
	}
	return "", fmt.Errorf("tencent AIGC task timed out after %v (task_id=%s)", timeout, taskID)
}

// TencentAIGCResult 生图结果
type TencentAIGCResult struct {
	ImageURL    string // 腾讯 CDN 临时 URL（有效期约 7 天）
	TaskID      string
	RawResponse string // 简要 JSON 摘要，用于日志
}

// GenerateImage 生成纯文生图（无参考图）。
func (a *TencentAIGCAdapter) GenerateImage(
	prompt, modelName, modelVersion, aspectRatio, resolution string,
) (*TencentAIGCResult, error) {
	return a.GenerateImageWithURLRefs(prompt, modelName, modelVersion, aspectRatio, resolution, nil)
}

const tencentImagePollTimeout = imageRequestTimeout

// GenerateImageWithURLRefs 生成图片，可附带可公开访问的参考图 URL 列表（GEM 2.5/3.0 支持）。
func (a *TencentAIGCAdapter) GenerateImageWithURLRefs(
	prompt, modelName, modelVersion, aspectRatio, resolution string,
	refURLs []string,
) (*TencentAIGCResult, error) {
	var fileInfos []tencentAIGCFileInfo
	for _, u := range refURLs {
		if u != "" {
			fileInfos = append(fileInfos, tencentAIGCFileInfo{Type: "Url", URL: u})
		}
	}

	req := tencentAIGCCreateReq{
		ModelName:    modelName,
		ModelVersion: modelVersion,
		Prompt:       prompt,
		FileInfos:    fileInfos,
		OutputConfig: &tencentAIGCOutputConfig{
			StorageMode: "Temporary",
			AspectRatio: aspectRatio,
			Resolution:  mapResolution(resolution),
		},
	}

	taskID, err := a.createTask(req)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	imageURL, err := a.pollTask(taskID, tencentImagePollTimeout)
	if err != nil {
		return nil, err
	}

	return &TencentAIGCResult{
		ImageURL:    imageURL,
		TaskID:      taskID,
		RawResponse: fmt.Sprintf(`{"task_id":%q,"image_url":%q}`, taskID, imageURL),
	}, nil
}
