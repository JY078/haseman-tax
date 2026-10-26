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
	"time"

	"advertising-image/backend/internal/storage"
)

const (
	MagnificImageBaseURL       = "https://api.magnific.com"
	magnificImageAPIPath       = "/v1/ai/text-to-image"
	magnificImagePollInterval  = 5 * time.Second
	magnificImagePollAttempts  = 120
	magnificMaxReferenceImages = 14
	magnificNanoBananaPro      = "nano-banana-pro"
	magnificNanoBananaProFlash = "nano-banana-pro-flash"
)

type MagnificReferenceImage struct {
	Image    string `json:"image"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type MagnificImageRequest struct {
	Prompt              string                   `json:"prompt"`
	WebhookURL          string                   `json:"webhook_url,omitempty"`
	ReferenceImages     []MagnificReferenceImage `json:"reference_images,omitempty"`
	AspectRatio         string                   `json:"aspect_ratio,omitempty"`
	Resolution          string                   `json:"resolution,omitempty"`
	UseGoogleSearchTool *bool                    `json:"use_google_search_tool,omitempty"`
}

type magnificImageResponse struct {
	Data struct {
		Generated []string `json:"generated"`
		TaskID    string   `json:"task_id"`
		Status    string   `json:"status"`
	} `json:"data"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

type MagnificImageAdapter struct {
	APIKey  string
	BaseURL string

	pollInterval time.Duration
	pollAttempts int
}

func NewMagnificImageAdapter(apiKey string, baseURL string) *MagnificImageAdapter {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = MagnificImageBaseURL
	}
	return &MagnificImageAdapter{
		APIKey:       strings.TrimSpace(apiKey),
		BaseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		pollInterval: magnificImagePollInterval,
		pollAttempts: magnificImagePollAttempts,
	}
}

func IsMagnificURL(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "magnific.com" || strings.HasSuffix(host, ".magnific.com")
}

func IsMagnificImageModel(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case magnificNanoBananaPro, magnificNanoBananaProFlash:
		return true
	default:
		return false
	}
}

func (a *MagnificImageAdapter) GenerateImage(ctx context.Context, prompt string, aspectRatio string, resolution string, modelName string) (*GeminiImageResult, error) {
	return a.generate(ctx, prompt, nil, aspectRatio, resolution, modelName)
}

func (a *MagnificImageAdapter) GenerateImageWithRefs(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string, modelName string) (*GeminiImageResult, error) {
	return a.generate(ctx, prompt, refs, aspectRatio, resolution, modelName)
}

func (a *MagnificImageAdapter) EditImage(ctx context.Context, prompt string, imageData string, mimeType string, modelName string, imageConfig ...string) (*GeminiImageResult, error) {
	refs := []RefImage{{Data: imageData, MimeType: mimeType}}
	var aspectRatio string
	var resolution string
	if len(imageConfig) > 0 {
		aspectRatio = imageConfig[0]
	}
	if len(imageConfig) > 1 {
		resolution = imageConfig[1]
	}
	return a.generate(ctx, prompt, refs, aspectRatio, resolution, modelName)
}

func (a *MagnificImageAdapter) GenerateImageWithRef(ctx context.Context, prompt string, refImageData string, refMimeType string, aspectRatio string, resolution string, modelName string) (*GeminiImageResult, error) {
	refs := []RefImage{{Data: refImageData, MimeType: refMimeType}}
	return a.generate(ctx, prompt, refs, aspectRatio, resolution, modelName)
}

func (a *MagnificImageAdapter) generate(ctx context.Context, prompt string, refs []RefImage, aspectRatio string, resolution string, modelName string) (*GeminiImageResult, error) {
	if a.APIKey == "" {
		return nil, fmt.Errorf("Magnific API key not configured")
	}
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	if !IsMagnificImageModel(modelName) {
		return nil, fmt.Errorf("unsupported Magnific image model: %s", modelName)
	}
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("Magnific image prompt is empty")
	}

	referenceImages, tempRefPaths, err := magnificReferenceImages(refs)
	if len(tempRefPaths) > 0 {
		defer scheduleMagnificTempRefCleanup(tempRefPaths)
	}
	if err != nil {
		return nil, err
	}

	req := MagnificImageRequest{
		Prompt:          prompt,
		ReferenceImages: referenceImages,
		AspectRatio:     strings.TrimSpace(aspectRatio),
		Resolution:      strings.TrimSpace(resolution),
	}
	if modelName == magnificNanoBananaProFlash {
		useGoogleSearchTool := false
		req.UseGoogleSearchTool = &useGoogleSearchTool
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := a.endpointURL(modelName)
	initial, rawResponse, err := a.doRequest(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	if imageURL := firstMagnificGeneratedImage(initial); imageURL != "" {
		return magnificImageResult(imageURL, rawResponse), nil
	}
	if strings.TrimSpace(initial.Data.TaskID) == "" {
		return nil, magnificResponseError("Magnific create task returned no task id", initial)
	}

	return a.pollResult(ctx, endpoint, initial.Data.TaskID)
}

func (a *MagnificImageAdapter) pollResult(ctx context.Context, endpoint string, taskID string) (*GeminiImageResult, error) {
	queryURL := strings.TrimRight(endpoint, "/") + "/" + url.PathEscape(taskID)
	for attempt := 0; attempt < a.pollAttempts; attempt++ {
		timer := time.NewTimer(a.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("Magnific image task canceled: %w", ctx.Err())
		case <-timer.C:
		}

		result, rawResponse, err := a.doRequest(ctx, http.MethodGet, queryURL, nil)
		if err != nil {
			return nil, fmt.Errorf("query Magnific image task: %w", err)
		}
		if imageURL := firstMagnificGeneratedImage(result); imageURL != "" {
			return magnificImageResult(imageURL, rawResponse), nil
		}

		switch strings.ToUpper(strings.TrimSpace(result.Data.Status)) {
		case "COMPLETED":
			return nil, fmt.Errorf("Magnific image task completed without an image: task_id=%s", taskID)
		case "FAILED", "ERROR", "CANCELLED", "CANCELED":
			return nil, magnificResponseError("Magnific image task failed", result)
		}
	}
	return nil, fmt.Errorf("Magnific image task timed out: task_id=%s", taskID)
}

func (a *MagnificImageAdapter) doRequest(ctx context.Context, method string, requestURL string, body []byte) (magnificImageResponse, []byte, error) {
	var result magnificImageResponse
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return result, nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("x-magnific-api-key", a.APIKey)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := imageHTTPClient.Do(httpReq)
	if err != nil {
		return result, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return result, respBody, fmt.Errorf("Magnific API error %d: %s", resp.StatusCode, redactLongStrings(respBody))
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return result, respBody, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, respBody, nil
}

func (a *MagnificImageAdapter) endpointURL(modelName string) string {
	baseURL := strings.TrimRight(a.BaseURL, "/")
	for _, suffix := range []string{"/" + magnificNanoBananaProFlash, "/" + magnificNanoBananaPro} {
		if strings.HasSuffix(strings.ToLower(baseURL), suffix) {
			baseURL = strings.TrimRight(baseURL[:len(baseURL)-len(suffix)], "/")
			break
		}
	}
	if !strings.HasSuffix(strings.ToLower(baseURL), magnificImageAPIPath) {
		baseURL += magnificImageAPIPath
	}
	return baseURL + "/" + modelName
}

func magnificReferenceImages(refs []RefImage) ([]MagnificReferenceImage, []string, error) {
	images := make([]MagnificReferenceImage, 0, min(len(refs), magnificMaxReferenceImages))
	tempRefPaths := make([]string, 0, min(len(refs), magnificMaxReferenceImages))
	for _, ref := range refs {
		if len(images) == magnificMaxReferenceImages {
			break
		}
		imageURL := strings.TrimSpace(ref.URL)
		mimeType := strings.TrimSpace(ref.MimeType)
		if imageURL == "" && strings.TrimSpace(ref.Data) != "" {
			if mimeType == "" {
				mimeType = "image/png"
			}
			var err error
			imageURL, err = storage.SaveTempRefImage(ref.Data, mimeType)
			if err != nil {
				return nil, tempRefPaths, fmt.Errorf("save Magnific reference image: %w", err)
			}
			tempRefPaths = append(tempRefPaths, imageURL)
		}
		if imageURL == "" {
			continue
		}
		images = append(images, MagnificReferenceImage{Image: imageURL, MimeType: mimeType})
	}
	return images, tempRefPaths, nil
}

func scheduleMagnificTempRefCleanup(paths []string) {
	paths = append([]string(nil), paths...)
	go func() {
		time.Sleep(storage.TempRefTTL)
		storage.DeleteTempRefFiles(paths)
	}()
}

func firstMagnificGeneratedImage(resp magnificImageResponse) string {
	for _, imageURL := range resp.Data.Generated {
		if imageURL = strings.TrimSpace(imageURL); imageURL != "" {
			return imageURL
		}
	}
	return ""
}

func magnificImageResult(imageURL string, rawResponse []byte) *GeminiImageResult {
	return &GeminiImageResult{
		ImageURL:    imageURL,
		RawResponse: redactLongStrings(rawResponse),
	}
}

func magnificResponseError(prefix string, resp magnificImageResponse) error {
	detail := strings.TrimSpace(resp.Error)
	if detail == "" {
		detail = strings.TrimSpace(resp.Message)
	}
	if detail == "" {
		detail = strings.TrimSpace(resp.Data.Status)
	}
	if detail == "" {
		return fmt.Errorf("%s", prefix)
	}
	return fmt.Errorf("%s: %s", prefix, detail)
}
