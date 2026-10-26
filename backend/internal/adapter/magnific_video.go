package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const MagnificVideoBaseURL = "https://api.magnific.com/v1/ai/video"

type MagnificVideoRequest struct {
	Prompt      string `json:"prompt"`
	WebhookURL  string `json:"webhook_url,omitempty"`
	Image       string `json:"image,omitempty"`
	ImageEnd    string `json:"image_end,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	CameraFixed bool   `json:"camera_fixed"`
	Seed        int    `json:"seed"`
}

type MagnificVideoResult struct {
	VideoURL    string
	TaskID      string
	Status      string
	RawResponse map[string]interface{}
}

type MagnificVideoAdapter struct {
	APIKey  string
	BaseURL string
}

func NewMagnificVideoAdapter(apiKey string) *MagnificVideoAdapter {
	return &MagnificVideoAdapter{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: MagnificVideoBaseURL,
	}
}

func (a *MagnificVideoAdapter) Generate(ctx context.Context, endpoint string, req MagnificVideoRequest) (*MagnificVideoResult, error) {
	if a.APIKey == "" {
		return nil, fmt.Errorf("Magnific API key not configured")
	}
	endpoint = strings.Trim(endpoint, "/")
	if endpoint == "" {
		return nil, fmt.Errorf("missing video endpoint")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.BaseURL, "/")+"/"+endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-magnific-api-key", a.APIKey)

	resp, err := imageHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &MagnificVideoResult{
		VideoURL:    firstStringPath(raw, "video_url", "url", "output.video_url", "output.url", "data.video_url", "data.url", "data.generated.0", "result.video_url", "result.url"),
		TaskID:      firstStringPath(raw, "task_id", "id", "output.task_id", "output.id", "data.task_id", "data.id", "result.task_id", "result.id"),
		Status:      firstStringPath(raw, "status", "state", "output.status", "output.state", "data.status", "data.state", "result.status", "result.state"),
		RawResponse: raw,
	}, nil
}

func (a *MagnificVideoAdapter) Query(ctx context.Context, modelSlug string, taskID string) (*MagnificVideoResult, error) {
	if a.APIKey == "" {
		return nil, fmt.Errorf("Magnific API key not configured")
	}
	modelSlug = strings.Trim(modelSlug, "/")
	taskID = strings.TrimSpace(taskID)
	if modelSlug == "" || taskID == "" {
		return nil, fmt.Errorf("missing video model or task id")
	}

	queryURL := strings.TrimRight(a.BaseURL, "/") + "/" + modelSlug + "/" + url.PathEscape(taskID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-magnific-api-key", a.APIKey)

	resp, err := imageHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &MagnificVideoResult{
		VideoURL:    firstStringPath(raw, "video_url", "url", "output.video_url", "output.url", "data.video_url", "data.url", "data.generated.0", "result.video_url", "result.url"),
		TaskID:      firstStringPath(raw, "task_id", "id", "output.task_id", "output.id", "data.task_id", "data.id", "result.task_id", "result.id"),
		Status:      firstStringPath(raw, "status", "state", "output.status", "output.state", "data.status", "data.state", "result.status", "result.state"),
		RawResponse: raw,
	}, nil
}

func firstStringPath(raw map[string]interface{}, paths ...string) string {
	for _, path := range paths {
		if value := stringPath(raw, path); value != "" {
			return value
		}
	}
	return ""
}

func stringPath(raw map[string]interface{}, path string) string {
	var current interface{} = raw
	for _, part := range strings.Split(path, ".") {
		if index, ok := numericPathPart(part); ok {
			list, ok := current.([]interface{})
			if !ok || index < 0 || index >= len(list) {
				return ""
			}
			current = list[index]
			continue
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = m[part]
	}
	if s, ok := current.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func numericPathPart(value string) (int, bool) {
	if value == "0" {
		return 0, true
	}
	return 0, false
}
