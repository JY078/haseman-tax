package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const GeminiBaseURL = "https://yunwu.ai"

// GeminiPart represents a part of a Gemini content item
type GeminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inlineData,omitempty"`
	FileData   *GeminiFileData   `json:"fileData,omitempty"`
	ImageURL   string            `json:"imageUrl,omitempty"`
	ImageURL2  string            `json:"image_url,omitempty"`
	URL        string            `json:"url,omitempty"`
}

// GeminiInlineData holds base64-encoded image data
type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GeminiFileData holds a file reference in requests or URL-based image output.
type GeminiFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri,omitempty"`
	FileURI2 string `json:"file_uri,omitempty"`
	URI      string `json:"uri,omitempty"`
	URL      string `json:"url,omitempty"`
}

// GeminiContent represents a content item in the Gemini request
type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiImageConfig controls image generation parameters
type GeminiImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"` // e.g. "16:9", "1:1", "9:16"
	ImageSize   string `json:"imageSize,omitempty"`   // e.g. "512", "1K", "2K", "4K"
}

// GeminiGenerationConfig holds generation configuration
type GeminiGenerationConfig struct {
	ResponseModalities []string           `json:"responseModalities"`
	ImageConfig        *GeminiImageConfig `json:"imageConfig,omitempty"`
}

// GeminiEditConfig holds generation configuration for image editing mode.
// Omits imageConfig entirely to avoid forcing aspect ratio changes on edited images.
type GeminiEditConfig struct {
	ResponseModalities []string `json:"responseModalities"`
}

// GeminiEditRequest is the request body for Gemini image editing
type GeminiEditRequest struct {
	Contents         []GeminiContent  `json:"contents"`
	GenerationConfig GeminiEditConfig `json:"generationConfig"`
}

// GeminiImageRequest is the request body for Gemini image generation
type GeminiImageRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
}

// GeminiResponseContent is a content block in the Gemini response
type GeminiResponseContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role"`
}

// GeminiCandidate is a single candidate in the Gemini response
type GeminiCandidate struct {
	Content      GeminiResponseContent `json:"content"`
	FinishReason string                `json:"finishReason"`
	Index        int                   `json:"index"`
}

// GeminiImageResponse is the response from Gemini image generation
type GeminiImageResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

func (p GeminiPart) outputURL() string {
	for _, value := range []string{p.ImageURL, p.ImageURL2, p.URL} {
		if clean := strings.TrimSpace(value); clean != "" {
			return clean
		}
	}
	if p.FileData == nil {
		return ""
	}
	for _, value := range []string{p.FileData.FileURI, p.FileData.FileURI2, p.FileData.URI, p.FileData.URL} {
		if clean := strings.TrimSpace(value); clean != "" {
			return clean
		}
	}
	return ""
}

func parseGeminiImageResult(respBody []byte) (*GeminiImageResult, error) {
	var geminiResp GeminiImageResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	result := &GeminiImageResult{RawResponse: redactLongStrings(respBody)}
	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				result.Text = part.Text
			}
			if part.InlineData != nil {
				result.ImageData = part.InlineData.Data
				result.MimeType = part.InlineData.MimeType
			}
			if imageURL := part.outputURL(); imageURL != "" {
				result.ImageURL = imageURL
				if result.MimeType == "" && part.FileData != nil {
					result.MimeType = part.FileData.MimeType
				}
			}
		}
	}
	if result.ImageData == "" && result.ImageURL == "" {
		result.ImageURL = findGeminiImageURL(respBody)
	}
	return result, nil
}

func findGeminiImageURL(respBody []byte) string {
	var payload interface{}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return ""
	}
	return findGeminiImageURLValue(payload)
}

func findGeminiImageURLValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		for _, key := range []string{"imageUrl", "image_url", "fileUri", "file_uri", "uri", "url"} {
			if raw, ok := v[key].(string); ok {
				if clean := strings.TrimSpace(raw); strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") {
					return clean
				}
			}
		}
		for _, raw := range v {
			if found := findGeminiImageURLValue(raw); found != "" {
				return found
			}
		}
	case []interface{}:
		for _, raw := range v {
			if found := findGeminiImageURLValue(raw); found != "" {
				return found
			}
		}
	}
	return ""
}

// GeminiImageResult is the parsed result with base64 image data
type GeminiImageResult struct {
	ImageData   string // base64-encoded image
	ImageURL    string // URL returned by the provider when present
	MimeType    string
	Text        string // any text in the response
	RawResponse string // full JSON response with long data replaced by placeholders
}

// GeminiAdapter handles image generation using the Gemini API
type GeminiAdapter struct {
	APIKey  string
	BaseURL string
	// RetryLogger is called after each failed attempt (before the next retry).
	// attempt is 1-based; maxAttempts is the total retry ceiling.
	// Set by callers who want retry events written to persistent storage.
	RetryLogger func(attempt, maxAttempts int, err error)
}

func NewGeminiAdapter(apiKey string, baseURL string) *GeminiAdapter {
	if baseURL == "" {
		baseURL = GeminiBaseURL
	}
	return &GeminiAdapter{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

// longStringRe matches JSON string values longer than 200 characters.
var longStringRe = regexp.MustCompile(`"([^"\\]{200,})"`)

// redactLongStrings replaces long base64/binary string values in a JSON body
// with a "[...长数据占位符...]" placeholder to keep logs readable.
func redactLongStrings(raw []byte) string {
	return longStringRe.ReplaceAllString(string(raw), `"[...长数据占位符...]"`)
}

func redactGeminiRequestURL(rawURL string) string {
	if queryStart := strings.IndexByte(rawURL, '?'); queryStart >= 0 {
		return rawURL[:queryStart]
	}
	return rawURL
}

func redactGeminiNetworkError(err error, apiKey string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if key := strings.TrimSpace(apiKey); key != "" {
		message = strings.ReplaceAll(message, key, "[REDACTED]")
	}
	return fmt.Errorf("%s", message)
}

const (
	maxImageRetries           = 2
	geminiImageAttemptTimeout = imageRequestTimeout
	geminiRetryBaseBackoff    = time.Second
	geminiImageRetryLogLabel  = "[Gemini生图]"
)

// attemptTimeout caps each individual Gemini image-generation API call.
// Some upstream image jobs on yunwu.ai regularly exceed several minutes,
// especially for edit/reference-based generation, so we allow up to 20 minutes
// per image attempt to match the current product expectation.
const attemptTimeout = geminiImageAttemptTimeout

// doImageRequest executes an HTTP POST to the Gemini API with the provided body.
// ctx should be the Gin request context; canceled when the client disconnects or
// the server WriteTimeout fires — which immediately aborts all in-flight retries.
// Each attempt is independently capped by attemptTimeout so a single slow API
// call never blocks the goroutine for the full httpClient.Timeout duration.
func (a *GeminiAdapter) doImageRequest(ctx context.Context, url string, body []byte) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= maxImageRetries; attempt++ {
		// Honor parent context cancellation during the sleep between retries
		if attempt > 1 {
			wait := time.Duration(attempt-1)*geminiRetryBaseBackoff + time.Duration(rand.IntN(5))*time.Second
			log.Printf("[Gemini生图] 第 %d/%d 次重试，等待 %v 后发起请求", attempt, maxImageRetries, wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, fmt.Errorf("请求已取消: %w", ctx.Err())
			}
		}

		// Cap each individual attempt independently of the global httpClient timeout
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		httpReq, err := http.NewRequestWithContext(attemptCtx, "POST", url, bytes.NewBuffer(body))
		if err != nil {
			cancel()
			return nil, fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		t0 := time.Now()
		log.Printf("%s 第 %d/%d 次请求开始 url=%s bodyLen=%d", geminiImageRetryLogLabel, attempt, maxImageRetries, redactGeminiRequestURL(url), len(body))
		resp, err := imageHTTPClient.Do(httpReq)
		elapsed := time.Since(t0)
		if err != nil {
			cancel()
			safeErr := redactGeminiNetworkError(err, a.APIKey)
			if attemptCtx.Err() == context.DeadlineExceeded {
				lastErr = fmt.Errorf("do request timeout after %v: %w", attemptTimeout, safeErr)
			} else {
				lastErr = fmt.Errorf("do request: %w", safeErr)
			}
			log.Printf("%s 第 %d/%d 次请求失败（网络错误）耗时=%v: %v", geminiImageRetryLogLabel, attempt, maxImageRetries, elapsed.Round(time.Millisecond), safeErr)
			if a.RetryLogger != nil {
				a.RetryLogger(attempt, maxImageRetries, lastErr)
			}
			// If the parent context was canceled, stop retrying immediately
			if ctx.Err() != nil {
				log.Printf("%s 父context已取消，停止重试: %v", geminiImageRetryLogLabel, ctx.Err())
				return nil, lastErr
			}
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel() // cancel after body is fully read
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			log.Printf("%s 第 %d/%d 次请求失败（读取响应体错误）耗时=%v: %v", geminiImageRetryLogLabel, attempt, maxImageRetries, elapsed.Round(time.Millisecond), err)
			if a.RetryLogger != nil {
				a.RetryLogger(attempt, maxImageRetries, lastErr)
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
			log.Printf("%s 第 %d/%d 次请求失败（HTTP %d）耗时=%v body=%s",
				geminiImageRetryLogLabel, attempt, maxImageRetries, resp.StatusCode, elapsed.Round(time.Millisecond), redactLongStrings(respBody))
			if a.RetryLogger != nil {
				a.RetryLogger(attempt, maxImageRetries, lastErr)
			}
			continue
		}

		log.Printf("%s 第 %d/%d 次请求成功 耗时=%v respLen=%d", geminiImageRetryLogLabel, attempt, maxImageRetries, elapsed.Round(time.Millisecond), len(respBody))
		return respBody, nil
	}
	return nil, lastErr
}

// GenerateImage sends an image generation request to Gemini
func (a *GeminiAdapter) GenerateImage(ctx context.Context, prompt string, aspectRatio string, resolution string, modelName string) (*GeminiImageResult, error) {
	if IsOpenLuxImageModel(modelName) {
		return NewOpenLuxImageAdapter(a.APIKey, a.BaseURL).GenerateImage(ctx, prompt, modelName, aspectRatio, resolution)
	}
	if IsMagnificURL(a.BaseURL) {
		return NewMagnificImageAdapter(a.APIKey, a.BaseURL).GenerateImage(ctx, prompt, aspectRatio, resolution, modelName)
	}
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	if modelName == "" {
		return nil, fmt.Errorf("未指定绘图模型，请在模型管理中配置默认绘图模型")
	}

	req := GeminiImageRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			ResponseModalities: []string{"IMAGE"},
			ImageConfig: &GeminiImageConfig{
				AspectRatio: aspectRatio,
				ImageSize:   resolution,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	// log.Printf("[GenerateImage] 请求体: %s", string(body))

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", a.BaseURL, modelName, a.APIKey)

	respBody, err := a.doImageRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}

	return parseGeminiImageResult(respBody)
}

// AnalyzeImage sends an image to Gemini and returns a text-only analysis.
// It uses TEXT-only response modality (no image generation) — much faster and cheaper.
func (a *GeminiAdapter) AnalyzeImage(prompt string, imageData string, mimeType string, modelName string) (string, error) {
	if modelName == "" {
		return "", fmt.Errorf("未指定分析模型，请在模型管理中配置默认绘图模型")
	}

	req := GeminiImageRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{
						InlineData: &GeminiInlineData{
							MimeType: mimeType,
							Data:     imageData,
						},
					},
					{Text: prompt},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			ResponseModalities: []string{"TEXT"},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", a.BaseURL, modelName, a.APIKey)

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
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

	var geminiResp GeminiImageResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				return part.Text, nil
			}
		}
	}
	return "", fmt.Errorf("no text in response")
}

// RefImage holds a single reference image (base64 + mime type)
type RefImage struct {
	Data     string
	MimeType string
	URL      string
}

// GenerateImageWithRefs generates an image with multiple reference images (up to 3)
func (a *GeminiAdapter) GenerateImageWithRefs(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string, modelName string) (*GeminiImageResult, error) {
	if IsOpenLuxImageModel(modelName) {
		return NewOpenLuxImageAdapter(a.APIKey, a.BaseURL).GenerateImageWithRefs(ctx, prompt, modelName, aspectRatio, resolution, refs)
	}
	if IsMagnificURL(a.BaseURL) {
		return NewMagnificImageAdapter(a.APIKey, a.BaseURL).GenerateImageWithRefs(ctx, prompt, refs, aspectRatio, resolution, modelName)
	}
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	if modelName == "" {
		return nil, fmt.Errorf("未指定绘图模型，请在模型管理中配置默认绘图模型")
	}

	// Build parts: all images first, then the text prompt
	parts := make([]GeminiPart, 0, len(refs)+1)
	for _, ref := range refs {
		file, err := geminiReferenceFile(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("准备第 %d 张参考图: %w", len(parts)+1, err)
		}
		parts = append(parts, GeminiPart{FileData: file})
	}
	parts = append(parts, GeminiPart{Text: prompt})

	req := GeminiImageRequest{
		Contents: []GeminiContent{
			{Role: "user", Parts: parts},
		},
		GenerationConfig: GeminiGenerationConfig{
			ResponseModalities: []string{"IMAGE"},
			ImageConfig: &GeminiImageConfig{
				AspectRatio: aspectRatio,
				ImageSize:   resolution,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	// log.Printf("[GenerateImageWithRefs] 请求体: %s", string(body))

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", a.BaseURL, modelName, a.APIKey)

	respBody, err := a.doImageRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}

	return parseGeminiImageResult(respBody)
}

// EditImage modifies an existing image using Gemini's image editing mode.
//
// Design notes:
//   - Uses responseModalities:["IMAGE"] only — this forces the model to output an
//     image rather than a text description of the edit.
//   - Does NOT include imageConfig (no forced aspect ratio) so original dimensions
//     are preserved.
//   - Prompts passed to this method must be purely descriptive ("result state"
//     language), NOT imperative/operation language. Imperative prompts like
//     "Remove X" can trigger MALFORMED_FUNCTION_CALL on proxy APIs; descriptive
//     prompts like "A photo identical to the reference but without X" do not.
func (a *GeminiAdapter) EditImage(ctx context.Context, prompt string, imageData string, mimeType string, modelName string, imageConfig ...string) (*GeminiImageResult, error) {
	if IsOpenLuxImageModel(modelName) {
		return nil, fmt.Errorf("OpenLux images/generations 接口不支持参考图")
	}
	if IsMagnificURL(a.BaseURL) {
		return NewMagnificImageAdapter(a.APIKey, a.BaseURL).EditImage(ctx, prompt, imageData, mimeType, modelName, imageConfig...)
	}
	if modelName == "" {
		return nil, fmt.Errorf("未指定绘图模型，请在模型管理中配置默认绘图模型")
	}

	file, err := geminiReferenceFile(ctx, RefImage{Data: imageData, MimeType: mimeType})
	if err != nil {
		return nil, err
	}

	req := GeminiEditRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{FileData: file},
					{Text: prompt},
				},
			},
		},
		GenerationConfig: GeminiEditConfig{
			ResponseModalities: []string{"IMAGE"},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", a.BaseURL, modelName, a.APIKey)

	respBody, err := a.doImageRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}

	return parseGeminiImageResult(respBody)
}

// GenerateImageWithRef generates an image with a reference image.
// If aspectRatio and resolution are both empty, imageConfig is omitted entirely
// so the model preserves the original image dimensions.
func (a *GeminiAdapter) GenerateImageWithRef(ctx context.Context, prompt string, refImageData string, refMimeType string, aspectRatio string, resolution string, modelName string) (*GeminiImageResult, error) {
	if IsOpenLuxImageModel(modelName) {
		if strings.TrimSpace(refImageData) != "" {
			return nil, fmt.Errorf("OpenLux 参考图必须提供 URL，不能使用 base64")
		}
		return NewOpenLuxImageAdapter(a.APIKey, a.BaseURL).GenerateImage(ctx, prompt, modelName, aspectRatio, resolution)
	}
	if IsMagnificURL(a.BaseURL) {
		return NewMagnificImageAdapter(a.APIKey, a.BaseURL).GenerateImageWithRef(ctx, prompt, refImageData, refMimeType, aspectRatio, resolution, modelName)
	}
	if modelName == "" {
		return nil, fmt.Errorf("未指定绘图模型，请在模型管理中配置默认绘图模型")
	}

	var imageConfig *GeminiImageConfig
	if aspectRatio != "" || resolution != "" {
		imageConfig = &GeminiImageConfig{
			AspectRatio: aspectRatio,
			ImageSize:   resolution,
		}
	}

	file, err := geminiReferenceFile(ctx, RefImage{Data: refImageData, MimeType: refMimeType})
	if err != nil {
		return nil, err
	}

	// Image first, then instruction — multimodal models attend better to the image
	// when it appears before the text prompt.
	req := GeminiImageRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{FileData: file},
					{Text: prompt},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			ResponseModalities: []string{"IMAGE"},
			ImageConfig:        imageConfig,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	// log.Printf("[GenerateImageWithRef] 请求体: %s", string(body))

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", a.BaseURL, modelName, a.APIKey)

	respBody, err := a.doImageRequest(ctx, url, body)
	if err != nil {
		return nil, err
	}

	return parseGeminiImageResult(respBody)
}
