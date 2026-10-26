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

const ArkVideoBaseURL = "https://ark.cn-beijing.volces.com/api/v3"

type ArkVideoContent struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *ArkVideoImageURL `json:"image_url,omitempty"`
	VideoURL *ArkVideoImageURL `json:"video_url,omitempty"`
	AudioURL *ArkVideoImageURL `json:"audio_url,omitempty"`
	Role     string            `json:"role,omitempty"`
}

type ArkVideoImageURL struct {
	URL string `json:"url"`
}

type ArkVideoRequest struct {
	Model         string            `json:"model"`
	Content       []ArkVideoContent `json:"content"`
	CallbackURL   string            `json:"callback_url,omitempty"`
	Resolution    string            `json:"resolution,omitempty"`
	Ratio         string            `json:"ratio,omitempty"`
	Duration      int               `json:"duration,omitempty"`
	GenerateAudio bool              `json:"generate_audio"`
	CameraFixed   bool              `json:"camera_fixed,omitempty"`
	Seed          int64             `json:"seed"`
	Watermark     bool              `json:"watermark"`
}

type ArkVideoError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ArkVideoOutput struct {
	VideoURL     string `json:"video_url"`
	LastFrameURL string `json:"last_frame_url"`
}

type ArkVideoResult struct {
	ID          string          `json:"id"`
	Model       string          `json:"model"`
	Status      string          `json:"status"`
	Content     ArkVideoOutput  `json:"content"`
	Error       *ArkVideoError  `json:"error,omitempty"`
	RawResponse json.RawMessage `json:"-"`
}

type ArkVideoAdapter struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewArkVideoAdapter(apiKey, baseURL string) *ArkVideoAdapter {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = ArkVideoBaseURL
	}
	return &ArkVideoAdapter{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  imageHTTPClient,
	}
}

func (a *ArkVideoAdapter) Create(ctx context.Context, req ArkVideoRequest) (*ArkVideoResult, error) {
	return a.do(ctx, http.MethodPost, "/contents/generations/tasks", req)
}

func (a *ArkVideoAdapter) Query(ctx context.Context, taskID string) (*ArkVideoResult, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("missing video task id")
	}
	return a.do(ctx, http.MethodGet, "/contents/generations/tasks/"+url.PathEscape(taskID), nil)
}

func (a *ArkVideoAdapter) do(ctx context.Context, method, path string, payload interface{}) (*ArkVideoResult, error) {
	if a.APIKey == "" {
		return nil, fmt.Errorf("Ark API key not configured")
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, a.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	client := a.Client
	if client == nil {
		client = imageHTTPClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Ark API error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result ArkVideoResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	result.RawResponse = append(json.RawMessage(nil), respBody...)
	return &result, nil
}
