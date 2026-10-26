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

const viduBaseURL = "https://yunwu.ai"

// ── 模型辅助 ───────────────────────────────────────────────────────────────────

// IsViduModel 判断模型名是否应走 VIDU 接口。
// 目前仅支持 viduq2 模型。
func IsViduModel(name string) bool {
	lower := strings.ToLower(name)
	return lower == "viduq2"
}

// mapViduResolution 将系统分辨率标识转换为 VIDU API 接受的格式。
func mapViduResolution(res string) string {
	switch res {
	case "2K":
		return "2K"
	case "4K":
		return "4K"
	default: // "", "512", "1K"
		return "1080p"
	}
}

// ── 请求结构 ──────────────────────────────────────────────────────────────────

type viduCreateReq struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Images      []string `json:"images,omitempty"`
	AspectRatio string   `json:"aspect_ratio,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`
}

// ── 响应结构 ──────────────────────────────────────────────────────────────────

type viduCreateResp struct {
	TaskID string `json:"task_id"`
	State  string `json:"state"`
}

type viduCreationItem struct {
	URL  string `json:"url"`
	Type string `json:"type"` // "image"
}

// viduQueryResp 兼容 VIDU 原生格式与 yunwu.ai 代理可能包装的格式。
type viduQueryResp struct {
	// VIDU 原生格式
	State     string             `json:"state"` // created|queueing|processing|success|failed
	Creations []viduCreationItem `json:"creations"`
	ErrMsg    string             `json:"err_msg"`

	// yunwu.ai 代理可能包装为 Tencent 风格
	Response struct {
		Status        string `json:"Status"`
		AigcImageTask struct {
			Status string `json:"Status"`
			Output struct {
				FileInfos []struct {
					FileUrl string `json:"FileUrl"`
				} `json:"FileInfos"`
			} `json:"Output"`
		} `json:"AigcImageTask"`
	} `json:"Response"`
}

// ── ViduResult ─────────────────────────────────────────────────────────────────

// ViduResult 生图结果。
type ViduResult struct {
	ImageURL    string
	TaskID      string
	RawResponse string
}

// ── Adapter ────────────────────────────────────────────────────────────────────

// ViduAdapter 对接 VIDU 异步生图接口（通过 yunwu.ai 代理）。
type ViduAdapter struct {
	APIKey  string
	BaseURL string
}

// NewViduAdapter 创建 VIDU 适配器实例。
func NewViduAdapter(apiKey, baseURL string) *ViduAdapter {
	if baseURL == "" {
		baseURL = viduBaseURL
	}
	return &ViduAdapter{APIKey: apiKey, BaseURL: baseURL}
}

// createTask 提交生图任务，返回 task_id。
func (a *ViduAdapter) createTask(req viduCreateReq) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", a.BaseURL+"/ent/v2/reference2image", bytes.NewBuffer(body))
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

	var createResp viduCreateResp
	if err := json.Unmarshal(respBody, &createResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if createResp.TaskID == "" {
		return "", fmt.Errorf("no task_id in response: %s", string(respBody))
	}
	return createResp.TaskID, nil
}

// pollTask 轮询任务状态直到完成或超时，返回图片 URL。
// 采用逐步增长的间隔：3s → 4s → … 最大 10s。
func (a *ViduAdapter) pollTask(taskID string, timeout time.Duration) (string, string, error) {
	deadline := time.Now().Add(timeout)
	interval := 3 * time.Second
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		if interval < 10*time.Second {
			interval += time.Second
		}

		url := fmt.Sprintf("%s/ent/v2/tasks/%s/creations", a.BaseURL, taskID)
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

		var queryResp viduQueryResp
		if jsonErr := json.Unmarshal(respBody, &queryResp); jsonErr != nil {
			continue
		}

		// 优先解析 VIDU 原生格式
		state := strings.ToLower(queryResp.State)

		// 兼容代理可能包装的 Tencent 风格响应
		if state == "" {
			proxied := queryResp.Response.AigcImageTask.Status
			if proxied == "" {
				proxied = queryResp.Response.Status
			}
			switch strings.ToUpper(proxied) {
			case "FINISH":
				state = "success"
			case "FAIL":
				state = "failed"
			case "PROCESSING":
				state = "processing"
			}
		}

		switch state {
		case "success":
			// 先尝试 VIDU creations 数组
			if len(queryResp.Creations) > 0 && queryResp.Creations[0].URL != "" {
				return queryResp.Creations[0].URL, string(respBody), nil
			}
			// 降级：Tencent 风格 FileInfos
			files := queryResp.Response.AigcImageTask.Output.FileInfos
			if len(files) > 0 && files[0].FileUrl != "" {
				return files[0].FileUrl, string(respBody), nil
			}
			return "", "", fmt.Errorf("task succeeded but no image URL found in response: %s", string(respBody))

		case "failed":
			errDetail := queryResp.ErrMsg
			if errDetail == "" {
				errDetail = string(respBody)
			}
			return "", "", fmt.Errorf("VIDU task failed (task_id=%s): %s", taskID, errDetail)
		}
		// created / queueing / processing → 继续轮询
	}
	return "", "", fmt.Errorf("VIDU task timed out after %v (task_id=%s)", timeout, taskID)
}

// GenerateImageWithRefs 生成图片，支持 0 至 7 张 base64 参考图。
// refImages 使用与 Gemini 相同的 RefImage 类型，直接复用。
func (a *ViduAdapter) GenerateImageWithRefs(
	prompt, model, aspectRatio, resolution string,
	refImages []RefImage,
) (*ViduResult, error) {
	// VIDU 要求参考图格式：data:{mime_type};base64,{base64_data}
	var images []string
	for _, img := range refImages {
		if img.Data == "" {
			continue
		}
		mimeType := img.MimeType
		if mimeType == "" {
			mimeType = "image/png"
		}
		images = append(images, fmt.Sprintf("data:%s;base64,%s", mimeType, img.Data))
	}

	req := viduCreateReq{
		Model:       model,
		Prompt:      prompt,
		Images:      images,
		AspectRatio: aspectRatio,
		Resolution:  mapViduResolution(resolution),
	}

	taskID, err := a.createTask(req)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	imageURL, rawResp, err := a.pollTask(taskID, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	return &ViduResult{
		ImageURL:    imageURL,
		TaskID:      taskID,
		RawResponse: rawResp,
	}, nil
}
