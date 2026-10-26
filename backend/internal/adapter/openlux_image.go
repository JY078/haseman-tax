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

const openLuxImageGenerationPath = "/api/v3/images/generations"
const openLuxSeedreamModel = "doubao-seedream-5-0-pro-260628"

type OpenLuxImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size"`
	ResponseFormat string `json:"response_format"`
	Watermark      bool   `json:"watermark"`
	Image          any    `json:"image,omitempty"`
}

type OpenLuxImageAdapter struct {
	APIKey  string
	BaseURL string
}

func IsOpenLuxImageModel(modelName string) bool {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(modelName)), "/")
	return parts[len(parts)-1] == openLuxSeedreamModel
}

func NewOpenLuxImageAdapter(apiKey, baseURL string) *OpenLuxImageAdapter {
	return &OpenLuxImageAdapter{APIKey: apiKey, BaseURL: strings.TrimSpace(baseURL)}
}

func (a *OpenLuxImageAdapter) GenerateImage(ctx context.Context, prompt, modelName, aspectRatio, resolution string) (*GeminiImageResult, error) {
	return a.generateImage(ctx, prompt, modelName, aspectRatio, resolution, nil)
}

func (a *OpenLuxImageAdapter) GenerateImageWithRefs(ctx context.Context, prompt, modelName, aspectRatio, resolution string, refs []RefImage) (*GeminiImageResult, error) {
	return a.generateImage(ctx, prompt, modelName, aspectRatio, resolution, refs)
}

func (a *OpenLuxImageAdapter) generateImage(ctx context.Context, prompt, modelName, aspectRatio, resolution string, refs []RefImage) (*GeminiImageResult, error) {
	if strings.TrimSpace(a.BaseURL) == "" {
		return nil, fmt.Errorf("未配置生图请求地址")
	}
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("未配置默认生图模型")
	}

	size, err := mapOpenLuxImageSize(aspectRatio, resolution)
	if err != nil {
		return nil, err
	}
	reqBody := OpenLuxImageRequest{
		Model:          modelName,
		Prompt:         prompt,
		Size:           size,
		ResponseFormat: "url",
		Watermark:      false,
	}
	if len(refs) > 10 {
		return nil, fmt.Errorf("OpenLux 参考图最多支持 10 张")
	}
	if len(refs) == 1 {
		if strings.TrimSpace(refs[0].URL) == "" {
			return nil, fmt.Errorf("OpenLux 参考图必须提供 URL")
		}
		reqBody.Image = strings.TrimSpace(refs[0].URL)
	} else if len(refs) > 1 {
		urls := make([]string, 0, len(refs))
		for _, ref := range refs {
			url := strings.TrimSpace(ref.URL)
			if url == "" {
				return nil, fmt.Errorf("OpenLux 多张参考图必须全部提供 URL")
			}
			urls = append(urls, url)
		}
		reqBody.Image = urls
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openLuxImageGenerationURL(a.BaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := imageHTTPClient.Do(httpReq)
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

	result, err := parseGPTImageResponse(respBody)
	if err != nil {
		return nil, err
	}
	return &GeminiImageResult{
		ImageURL:    result.ImageURL,
		ImageData:   result.ImageData,
		MimeType:    result.MimeType,
		RawResponse: result.RawResponse,
	}, nil
}

func openLuxImageGenerationURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(strings.ToLower(baseURL), openLuxImageGenerationPath) {
		return baseURL
	}
	return baseURL + openLuxImageGenerationPath
}

func mapOpenLuxImageSize(aspectRatio, resolution string) (string, error) {
	aspectRatio = strings.TrimSpace(aspectRatio)
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	sizes := map[string]map[string]string{
		"1K": {
			"1:1": "1024x1024", "4:3": "1152x864", "3:4": "864x1152", "16:9": "1424x800",
			"9:16": "800x1424", "3:2": "1248x832", "2:3": "832x1248", "21:9": "1568x672",
		},
		"2K": {
			"1:1": "2048x2048", "4:3": "2368x1776", "3:4": "1776x2368", "16:9": "2816x1584",
			"9:16": "1584x2816", "3:2": "2496x1664", "2:3": "1664x2496", "21:9": "3136x1344",
		},
	}
	quality := strings.ToUpper(strings.TrimSpace(resolution))
	if quality != "1K" && quality != "2K" {
		return "", fmt.Errorf("Doubao 5.0 只支持 1K 或 2K 清晰度")
	}
	size, ok := sizes[quality][aspectRatio]
	if !ok {
		return "", fmt.Errorf("Doubao 5.0 不支持图片比例 %s", aspectRatio)
	}
	return size, nil
}
