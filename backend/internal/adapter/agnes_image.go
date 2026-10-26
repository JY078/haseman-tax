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

const (
	AgnesImageModel   = "agnes-image-2.1-flash"
	AgnesImageBaseURL = "https://apihub.agnes-ai.com"
	AgnesImageAPIKey  = "sk-TvGQZIBsWi9Bsd0RZDrLfRX8aC0YzIAcpTRcvBqgM2N1fzup"
)

type AgnesImageRequest struct {
	Model        string                 `json:"model"`
	Prompt       string                 `json:"prompt"`
	Size         string                 `json:"size,omitempty"`
	ReturnBase64 bool                   `json:"return_base64,omitempty"`
	ExtraBody    map[string]interface{} `json:"extra_body,omitempty"`
}

type AgnesImageData struct {
	B64JSON string `json:"b64_json"`
	URL     string `json:"url"`
}

type AgnesImageResponse struct {
	Data []AgnesImageData `json:"data"`
}

type AgnesImageAdapter struct {
	APIKey  string
	BaseURL string
}

func IsAgnesImageModel(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), AgnesImageModel)
}

func NewAgnesImageAdapter() *AgnesImageAdapter {
	return &AgnesImageAdapter{APIKey: AgnesImageAPIKey, BaseURL: AgnesImageBaseURL}
}

func (a *AgnesImageAdapter) GenerateImage(ctx context.Context, prompt string, aspectRatio string, resolution string) (*ApizGPTImageResult, error) {
	return a.generate(ctx, prompt, nil, aspectRatio, resolution)
}

func (a *AgnesImageAdapter) EditImage(ctx context.Context, prompt string, imageData string, mimeType string, aspectRatio string, resolution string) (*ApizGPTImageResult, error) {
	return a.EditImages(ctx, prompt, []RefImage{{Data: imageData, MimeType: mimeType}}, aspectRatio, resolution)
}

func (a *AgnesImageAdapter) EditImages(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string) (*ApizGPTImageResult, error) {
	return a.generate(ctx, prompt, refs, aspectRatio, resolution)
}

func (a *AgnesImageAdapter) generate(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string) (*ApizGPTImageResult, error) {
	if strings.TrimSpace(a.APIKey) == "" {
		return nil, fmt.Errorf("Agnes API key not configured")
	}

	params := MapGPTImageParams(aspectRatio, resolution)
	// 保持 Agnes 原有的方形 4K 尺寸，不跟随 GPT 的尺寸调整。
	if params.Size == "4096x4096" {
		params.Size = "2560x2560"
	}
	imageURLs, err := agnesRefImageURLs(refs)
	if err != nil {
		return nil, err
	}

	reqBody := AgnesImageRequest{
		Model:  AgnesImageModel,
		Prompt: prompt,
		Size:   params.Size,
	}
	if len(imageURLs) > 0 {
		reqBody.ExtraBody = map[string]interface{}{
			"image":           imageURLs,
			"response_format": "b64_json",
		}
	} else {
		reqBody.ReturnBase64 = true
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/v1/images/generations", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	log.Printf("[Agnes Image] start request model=%s size=%s refCount=%d", AgnesImageModel, params.Size, len(imageURLs))
	t0 := time.Now()
	resp, err := imageHTTPClient.Do(httpReq)
	log.Printf("[Agnes Image] request finished model=%s elapsed=%v", AgnesImageModel, time.Since(t0).Round(time.Millisecond))
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return parseAgnesImageResponse(respBody)
}

func agnesRefImageURLs(refs []RefImage) ([]string, error) {
	imageURLs := make([]string, 0, len(refs))
	for _, ref := range refs {
		if refURL := strings.TrimSpace(ref.URL); refURL != "" {
			imageURLs = append(imageURLs, refURL)
			continue
		}
		if strings.TrimSpace(ref.Data) == "" {
			continue
		}
		mimeType := strings.TrimSpace(ref.MimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		ossURL, err := storage.SaveTempRefImage(ref.Data, mimeType)
		if err != nil {
			return nil, fmt.Errorf("save reference image: %w", err)
		}
		imageURLs = append(imageURLs, ossURL)
	}
	return imageURLs, nil
}

func parseAgnesImageResponse(respBody []byte) (*ApizGPTImageResult, error) {
	var parsed AgnesImageResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	for _, item := range parsed.Data {
		if b64 := strings.TrimSpace(item.B64JSON); b64 != "" {
			return &ApizGPTImageResult{ImageData: b64, MimeType: "image/png"}, nil
		}
		if imageURL := strings.TrimSpace(item.URL); imageURL != "" {
			return &ApizGPTImageResult{ImageURL: imageURL, MimeType: "image/png"}, nil
		}
	}
	return nil, fmt.Errorf("no b64_json or url in response")
}
