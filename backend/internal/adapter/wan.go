package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const WanBaseURL = "https://yunwu.ai"

func IsWanModel(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(lower, "wan")
}

type WanGenerateRequest struct {
	Model        string   `json:"model"`
	Prompt       string   `json:"prompt"`
	Size         string   `json:"size,omitempty"`
	N            int      `json:"n,omitempty"`
	Watermark    bool     `json:"watermark,omitempty"`
	PromptExtend bool     `json:"prompt_extend,omitempty"`
	Images       []string `json:"images,omitempty"` // base64 encoded reference images (支持多个)
}

type WanData struct {
	B64JSON string `json:"b64_json"`
	URL     string `json:"url"`
}

type WanResponse struct {
	Created int64    `json:"created"`
	Data    []WanData `json:"data"`
}

type WanResult struct {
	ImageData   string
	MimeType    string
	ImageURL    string
	RawResponse string
}

type WanVariant struct {
	ImageData string
	MimeType  string
	ImageURL  string
}

type WanBatchResult struct {
	Items       []WanVariant
	RawResponse string
}

type WanAdapter struct {
	APIKey  string
	BaseURL string
}

func NewWanAdapter(apiKey string, baseURL string) *WanAdapter {
	if baseURL == "" {
		baseURL = WanBaseURL
	}
	return &WanAdapter{APIKey: apiKey, BaseURL: baseURL}
}

func MapWanSize(aspectRatio string, resolution string) string {
	is2K := strings.EqualFold(strings.TrimSpace(resolution), "2K")
	if is2K {
		switch strings.TrimSpace(aspectRatio) {
		case "3:4":
			return "1536x2048"
		case "9:16":
			return "1152x2048"
		case "4:3":
			return "2048x1536"
		case "16:9":
			return "2048x1152"
		case "21:9":
			return "2560x1080"
		default:
			return "2048x2048"
		}
	}
	switch strings.TrimSpace(aspectRatio) {
	case "3:4", "9:16":
		return "1024x1536"
	case "4:3", "16:9", "21:9":
		return "1536x1024"
	default:
		return "1024x1024"
	}
}

func (a *WanAdapter) GenerateImage(ctx context.Context, prompt string, modelName string, aspectRatio string, resolution string, n int, watermark bool, promptExtend bool, imageData []string) (*WanResult, error) {
	batch, err := a.GenerateImages(ctx, prompt, modelName, aspectRatio, resolution, n, watermark, promptExtend, imageData)
	if err != nil {
		return nil, err
	}
	if len(batch.Items) == 0 {
		return nil, fmt.Errorf("no image data in response")
	}
	first := batch.Items[0]
	return &WanResult{
		ImageData:   first.ImageData,
		MimeType:    first.MimeType,
		ImageURL:    first.ImageURL,
		RawResponse: batch.RawResponse,
	}, nil
}

func (a *WanAdapter) GenerateImages(ctx context.Context, prompt string, modelName string, aspectRatio string, resolution string, n int, watermark bool, promptExtend bool, imageData []string) (*WanBatchResult, error) {
	if modelName == "" {
		return nil, fmt.Errorf("未指定绘图模型，请在模型管理中配置默认绘图模型")
	}
	if n <= 0 {
		n = 1
	}

	items := make([]WanVariant, 0, n)
	rawResponses := make([]string, 0, n)
	for i := 0; i < n; i++ {
		item, rawResponse, err := a.generateSingleImage(ctx, prompt, modelName, aspectRatio, resolution, watermark, promptExtend, imageData)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		rawResponses = append(rawResponses, rawResponse)
	}

	return &WanBatchResult{Items: items, RawResponse: strings.Join(rawResponses, "\n")}, nil
}

func (a *WanAdapter) generateSingleImage(ctx context.Context, prompt string, modelName string, aspectRatio string, resolution string, watermark bool, promptExtend bool, imageData []string) (WanVariant, string, error) {
	reqBody := WanGenerateRequest{
		Model:        modelName,
		Prompt:       prompt,
		Size:         MapWanSize(aspectRatio, resolution),
		N:            1,
		Watermark:    watermark,
		PromptExtend: promptExtend,
		Images:       imageData,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return WanVariant{}, "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/v1/images/generations", bytes.NewBuffer(body))
	if err != nil {
		return WanVariant{}, "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := imageHTTPClient.Do(httpReq)
	if err != nil {
		return WanVariant{}, "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return WanVariant{}, "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return WanVariant{}, "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	batch, err := parseWanBatchResponse(respBody)
	if err != nil {
		return WanVariant{}, "", err
	}
	if len(batch.Items) == 0 {
		return WanVariant{}, "", fmt.Errorf("no image data in response")
	}
	return batch.Items[0], batch.RawResponse, nil
}

func parseWanBatchResponse(respBody []byte) (*WanBatchResult, error) {
	var parsed WanResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("no image data in response")
	}

	items := make([]WanVariant, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		result := WanVariant{MimeType: "image/png"}
		if strings.TrimSpace(item.B64JSON) != "" {
			result.ImageData = strings.TrimSpace(item.B64JSON)
			items = append(items, result)
			continue
		}
		if strings.TrimSpace(item.URL) != "" {
			result.ImageURL = strings.TrimSpace(item.URL)
			items = append(items, result)
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no b64_json or url in response")
	}

	return &WanBatchResult{Items: items, RawResponse: string(respBody)}, nil
}
