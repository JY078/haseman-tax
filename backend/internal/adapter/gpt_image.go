package adapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const GPTImageBaseURL = "https://yunwu.ai"

func IsGPTImageModel(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	parts := strings.Split(lower, "/")
	return strings.HasPrefix(parts[len(parts)-1], "gpt-image-")
}

type GPTImageGenerateRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	N              int    `json:"n,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type GPTImageData struct {
	B64JSON string `json:"b64_json"`
	URL     string `json:"url"`
}

type GPTImageResponse struct {
	Created int64          `json:"created"`
	Data    []GPTImageData `json:"data"`
}

type GPTImageResult struct {
	ImageData   string
	MimeType    string
	ImageURL    string
	RawResponse string
}

type GPTImageVariant struct {
	ImageData string
	MimeType  string
	ImageURL  string
}

type GPTImageBatchResult struct {
	Items       []GPTImageVariant
	RawResponse string
}

type GPTImageAdapter struct {
	APIKey       string
	BaseURL      string
	OutputWidth  int
	OutputHeight int
}

type imageEditAPIError struct {
	status int
	body   string
}

func (e *imageEditAPIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.status, e.body)
}

func NewGPTImageAdapter(apiKey string, baseURL string) *GPTImageAdapter {
	if baseURL == "" {
		baseURL = GPTImageBaseURL
	}
	return &GPTImageAdapter{APIKey: apiKey, BaseURL: baseURL}
}

// GPTImageParams 包含 GPT 图片生成所需的参数
type GPTImageParams struct {
	Size    string
	Quality string
}

// MapGPTImageParams 将 aspect ratio 和 resolution 映射为 GPT 图片生成参数
// 非 1:1 + 4K 的原有尺寸规则（1:1 + 4K 使用 4096x4096）:
//   - 最大边长 ≤ 3840px
//   - 宽高均为 16px 的倍数
//   - 长边/短边 ≤ 3:1
//   - 总像素: 655360 ~ 8294400
//
// resolution 映射: 1K→low, 2K→medium, 4K→high
func MapGPTImageParams(aspectRatio string, resolution string) GPTImageParams {
	aspectRatio = strings.TrimSpace(aspectRatio)
	resolution = strings.TrimSpace(resolution)
	is2K := strings.EqualFold(resolution, "2K")
	is4K := strings.EqualFold(resolution, "4K")

	var size string
	var quality string

	// 根据分辨率确定质量: 1K→low, 2K→medium, 4K→high
	if is4K {
		quality = "high"
	} else if is2K {
		quality = "medium"
	} else {
		quality = "low"
	}

	// 根据宽高比和分辨率确定尺寸
	// 1:1 + 4K 使用 4096x4096，其余比例保持原有映射。
	switch aspectRatio {
	case "1:1":
		if is4K {
			size = "4096x4096" // 总像素 16777216
		} else if is2K {
			size = "2048x2048" // 总像素 4194304
		} else {
			size = "1024x1024" // 总像素 1048576
		}
	case "4:3":
		if is4K {
			size = "2880x2160" // 总像素 6220800
		} else if is2K {
			size = "2048x1536" // 总像素 3145728
		} else {
			size = "1536x1152" // 总像素 1769472
		}
	case "3:4":
		if is4K {
			size = "2160x2880" // 总像素 6220800
		} else if is2K {
			size = "1536x2048" // 总像素 3145728
		} else {
			size = "1152x1536" // 总像素 1769472
		}
	case "16:9":
		if is4K {
			size = "3200x1800" // 总像素 5760000
		} else if is2K {
			size = "2048x1152" // 总像素 2359296
		} else {
			size = "1536x864" // 总像素 1327104
		}
	case "9:16":
		if is4K {
			size = "1800x3200" // 总像素 5760000
		} else if is2K {
			size = "1152x2048" // 总像素 2359296
		} else {
			size = "864x1536" // 总像素 1327104
		}
	case "4:5":
		if is4K {
			size = "2560x3200" // 总像素 8192000
		} else if is2K {
			size = "1664x2080" // 总像素 3461120
		} else {
			size = "832x1040" // 总像素 865280
		}
	case "5:4":
		if is4K {
			size = "3200x2560" // 总像素 8192000
		} else if is2K {
			size = "2080x1664" // 总像素 3461120
		} else {
			size = "1040x832" // 总像素 865280
		}
	case "21:9":
		if is4K {
			size = "3360x1440" // 总像素 4838400
		} else if is2K {
			size = "2928x1200" // 总像素 3513600, 比例=2.44:1<3:1 ✓
		} else {
			size = "1504x624" // 总像素 938496, 比例=2.41:1<3:1 ✓
		}
	case "2:1", "1:2", "3:2", "2:3", "9:21", "4:1", "1:4", "8:1", "1:8":
		size = mapExtendedGPTImageSize(aspectRatio, is2K, is4K)
	default:
		// 默认 1:1
		if is4K {
			size = "4096x4096"
		} else if is2K {
			size = "2048x2048"
		} else {
			size = "1024x1024"
		}
	}

	return GPTImageParams{Size: size, Quality: quality}
}

func mapExtendedGPTImageSize(aspectRatio string, is2K bool, is4K bool) string {
	sizes := map[string][3]string{
		"2:1":  {"1408x704", "2304x1152", "3200x1600"},
		"1:2":  {"704x1408", "1152x2304", "1600x3200"},
		"3:2":  {"1344x896", "2112x1408", "2880x1920"},
		"2:3":  {"896x1344", "1408x2112", "1920x2880"},
		"9:21": {"624x1504", "864x2016", "1440x3360"},
		"4:1":  {"1600x400", "2560x640", "3840x960"},
		"1:4":  {"400x1600", "640x2560", "960x3840"},
		"8:1":  {"1600x200", "2560x320", "3840x480"},
		"1:8":  {"200x1600", "320x2560", "480x3840"},
	}
	values, ok := sizes[aspectRatio]
	if !ok {
		return "1024x1024"
	}
	if is4K {
		return values[2]
	}
	if is2K {
		return values[1]
	}
	return values[0]
}

func (a *GPTImageAdapter) GenerateImage(ctx context.Context, prompt string, modelName string, aspectRatio string, resolution string, n int) (*GPTImageResult, error) {
	batch, err := a.GenerateImages(ctx, prompt, modelName, aspectRatio, resolution, n)
	if err != nil {
		return nil, err
	}
	if len(batch.Items) == 0 {
		return nil, fmt.Errorf("no image data in response")
	}
	first := batch.Items[0]
	return &GPTImageResult{
		ImageData:   first.ImageData,
		MimeType:    first.MimeType,
		ImageURL:    first.ImageURL,
		RawResponse: batch.RawResponse,
	}, nil
}

func (a *GPTImageAdapter) GenerateImages(ctx context.Context, prompt string, modelName string, aspectRatio string, resolution string, n int) (*GPTImageBatchResult, error) {
	if modelName == "" {
		return nil, fmt.Errorf("未指定绘图模型，请在模型管理中配置默认绘图模型")
	}
	if n <= 0 {
		n = 1
	}

	items := make([]GPTImageVariant, 0, n)
	rawResponses := make([]string, 0, n)
	for i := 0; i < n; i++ {
		batch, err := a.generateImages(ctx, prompt, modelName, aspectRatio, resolution, 1)
		if err != nil {
			return nil, err
		}
		if len(batch.Items) == 0 {
			return nil, fmt.Errorf("no image data in response")
		}
		items = append(items, batch.Items[0])
		rawResponses = append(rawResponses, batch.RawResponse)
	}
	return &GPTImageBatchResult{Items: items, RawResponse: strings.Join(rawResponses, "\n")}, nil
}

func (a *GPTImageAdapter) generateImages(ctx context.Context, prompt string, modelName string, aspectRatio string, resolution string, n int) (*GPTImageBatchResult, error) {
	params := a.imageParams(modelName, aspectRatio, resolution)
	reqBody := GPTImageGenerateRequest{
		Model:          modelName,
		Prompt:         prompt,
		Size:           params.Size,
		Quality:        params.Quality,
		N:              n,
		ResponseFormat: "url",
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

	log.Printf("[GPT生图] 开始调用API model=%s size=%s quality=%s", modelName, params.Size, params.Quality)
	t0 := time.Now()
	resp, err := imageHTTPClient.Do(httpReq)
	log.Printf("[GPT生图] API调用完成 model=%s 耗时=%v", modelName, time.Since(t0).Round(time.Millisecond))
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

	batch, err := parseGPTImageBatchResponse(respBody)
	if err != nil {
		return nil, err
	}
	log.Printf("[GPT生图] generations response model=%s requested=%d returned=%d", modelName, n, len(batch.Items))
	return batch, nil
}

func (a *GPTImageAdapter) EditImage(ctx context.Context, prompt string, refs []RefImage, modelName string, aspectRatio string, resolution string, n int) (*GPTImageResult, error) {
	batch, err := a.EditImages(ctx, prompt, refs, modelName, aspectRatio, resolution, n)
	if err != nil {
		return nil, err
	}
	if len(batch.Items) == 0 {
		return nil, fmt.Errorf("no image data in response")
	}
	first := batch.Items[0]
	return &GPTImageResult{
		ImageData:   first.ImageData,
		MimeType:    first.MimeType,
		ImageURL:    first.ImageURL,
		RawResponse: batch.RawResponse,
	}, nil
}

func (a *GPTImageAdapter) EditImages(ctx context.Context, prompt string, refs []RefImage, modelName string, aspectRatio string, resolution string, n int) (*GPTImageBatchResult, error) {
	if modelName == "" {
		return nil, fmt.Errorf("未指定绘图模型，请在模型管理中配置默认绘图模型")
	}
	if len(refs) == 0 {
		return a.GenerateImages(ctx, prompt, modelName, aspectRatio, resolution, n)
	}
	if n <= 0 {
		n = 1
	}

	items := make([]GPTImageVariant, 0, n)
	rawResponses := make([]string, 0, n)
	for i := 0; i < n; i++ {
		batch, err := a.editImagesBuffered(ctx, prompt, refs, modelName, aspectRatio, resolution, 1)
		for retry := 0; modelName == "flux.1-kontext-pro" && retry < 2; retry++ {
			var apiErr *imageEditAPIError
			if !errors.As(err, &apiErr) || apiErr.status != http.StatusServiceUnavailable {
				break
			}
			delay := time.Duration(2<<retry) * time.Second
			log.Printf("[Flux编辑] 上游503，%v后重试 (%d/2)", delay, retry+1)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
			batch, err = a.editImagesBuffered(ctx, prompt, refs, modelName, aspectRatio, resolution, 1)
		}
		if err != nil {
			return nil, err
		}
		if len(batch.Items) == 0 {
			return nil, fmt.Errorf("no image data in response")
		}
		items = append(items, batch.Items[0])
		rawResponses = append(rawResponses, batch.RawResponse)
	}
	return &GPTImageBatchResult{Items: items, RawResponse: strings.Join(rawResponses, "\n")}, nil
}

func (a *GPTImageAdapter) editSingleImage(ctx context.Context, prompt string, refs []RefImage, modelName string, aspectRatio string, resolution string) (GPTImageVariant, string, error) {
	log.Printf("[GPT生图] 开始调用EditImage API model=%s refCount=%d", modelName, len(refs))
	t0 := time.Now()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for idx, ref := range refs {
		if strings.TrimSpace(ref.Data) == "" {
			continue
		}
		mimeType := strings.TrimSpace(ref.MimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		ext := extFromMimeType(mimeType)
		part, err := writer.CreateFormFile("image", fmt.Sprintf("ref-%d%s", idx+1, ext))
		if err != nil {
			return GPTImageVariant{}, "", fmt.Errorf("create image form field: %w", err)
		}
		bin, err := decodeBase64String(ref.Data)
		if err != nil {
			return GPTImageVariant{}, "", fmt.Errorf("decode ref image: %w", err)
		}
		if _, err := part.Write(bin); err != nil {
			return GPTImageVariant{}, "", fmt.Errorf("write ref image: %w", err)
		}
	}
	_ = writer.WriteField("prompt", prompt)
	_ = writer.WriteField("model", modelName)
	_ = writer.WriteField("n", "1")
	params := a.imageParams(modelName, aspectRatio, resolution)
	_ = writer.WriteField("size", params.Size)
	_ = writer.WriteField("quality", params.Quality)
	if err := writer.Close(); err != nil {
		return GPTImageVariant{}, "", fmt.Errorf("close multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/v1/images/edits", &body)
	if err != nil {
		return GPTImageVariant{}, "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := imageHTTPClient.Do(httpReq)
	log.Printf("[GPT生图] EditImage API调用完成 model=%s 耗时=%v", modelName, time.Since(t0).Round(time.Millisecond))
	if err != nil {
		return GPTImageVariant{}, "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GPTImageVariant{}, "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return GPTImageVariant{}, "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	batch, err := parseGPTImageBatchResponse(respBody)
	if err != nil {
		return GPTImageVariant{}, "", err
	}
	if len(batch.Items) == 0 {
		return GPTImageVariant{}, "", fmt.Errorf("no image data in response")
	}
	return batch.Items[0], batch.RawResponse, nil
}

func (a *GPTImageAdapter) editImagesBuffered(ctx context.Context, prompt string, refs []RefImage, modelName string, aspectRatio string, resolution string, n int) (*GPTImageBatchResult, error) {
	log.Printf("[GPT生图] 开始调用EditImage API model=%s refCount=%d", modelName, len(refs))
	t0 := time.Now()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	params := a.imageParams(modelName, aspectRatio, resolution)
	if err := writeGPTImageEditMultipart(ctx, writer, prompt, refs, modelName, params, n); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/v1/images/edits", &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)

	resp, err := imageHTTPClient.Do(httpReq)
	log.Printf("[GPT生图] EditImage API调用完成 model=%s 耗时=%v", modelName, time.Since(t0).Round(time.Millisecond))
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &imageEditAPIError{status: resp.StatusCode, body: string(respBody)}
	}

	batch, err := parseGPTImageBatchResponse(respBody)
	if err != nil {
		return nil, err
	}
	log.Printf("[GPT生图] edits response model=%s requested=%d returned=%d", modelName, n, len(batch.Items))
	return batch, nil
}

func writeGPTImageEditMultipart(ctx context.Context, writer *multipart.Writer, prompt string, refs []RefImage, modelName string, params GPTImageParams, n int) error {
	wroteImages := 0
	for idx, ref := range refs {
		refURL := strings.TrimSpace(ref.URL)
		mimeType := strings.TrimSpace(ref.MimeType)
		if refURL != "" {
			if mimeType == "" {
				mimeType = "image/png"
			}
			part, err := writer.CreateFormFile("image", fmt.Sprintf("ref-%d%s", idx+1, extFromMimeType(mimeType)))
			if err != nil {
				return fmt.Errorf("create image form field: %w", err)
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, refURL, nil)
			if err != nil {
				return fmt.Errorf("create ref download request: %w", err)
			}
			resp, err := imageHTTPClient.Do(req)
			if err != nil {
				return fmt.Errorf("download ref image: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return fmt.Errorf("download ref image: HTTP %d", resp.StatusCode)
			}
			if _, err := io.Copy(part, resp.Body); err != nil {
				resp.Body.Close()
				return fmt.Errorf("write ref image: %w", err)
			}
			resp.Body.Close()
			wroteImages++
			continue
		}

		if strings.TrimSpace(ref.Data) == "" {
			continue
		}
		if mimeType == "" {
			mimeType = "image/png"
		}
		part, err := writer.CreateFormFile("image", fmt.Sprintf("ref-%d%s", idx+1, extFromMimeType(mimeType)))
		if err != nil {
			return fmt.Errorf("create image form field: %w", err)
		}
		bin, err := decodeBase64String(ref.Data)
		if err != nil {
			return fmt.Errorf("decode ref image: %w", err)
		}
		if _, err := part.Write(bin); err != nil {
			return fmt.Errorf("write ref image: %w", err)
		}
		wroteImages++
	}
	if wroteImages == 0 {
		return fmt.Errorf("no reference image data")
	}
	_ = writer.WriteField("prompt", prompt)
	_ = writer.WriteField("model", modelName)
	_ = writer.WriteField("n", fmt.Sprintf("%d", n))
	_ = writer.WriteField("size", params.Size)
	if modelName != "flux.1-kontext-pro" {
		_ = writer.WriteField("quality", params.Quality)
	}
	return writer.Close()
}

func parseGPTImageResponse(respBody []byte) (*GPTImageResult, error) {
	batch, err := parseGPTImageBatchResponse(respBody)
	if err != nil {
		return nil, err
	}
	if len(batch.Items) == 0 {
		return nil, fmt.Errorf("no image data in response")
	}
	first := batch.Items[0]
	return &GPTImageResult{
		ImageData:   first.ImageData,
		MimeType:    first.MimeType,
		ImageURL:    first.ImageURL,
		RawResponse: batch.RawResponse,
	}, nil
}

func parseGPTImageBatchResponse(respBody []byte) (*GPTImageBatchResult, error) {
	var parsed GPTImageResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("no image data in response")
	}

	items := make([]GPTImageVariant, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		result := GPTImageVariant{MimeType: "image/png"}
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

	return &GPTImageBatchResult{Items: items, RawResponse: string(respBody)}, nil
}

func decodeBase64String(data string) ([]byte, error) {
	clean := strings.TrimSpace(data)
	if idx := strings.Index(clean, ","); idx >= 0 {
		clean = clean[idx+1:]
	}
	bin, err := base64.StdEncoding.DecodeString(clean)
	if err == nil {
		return bin, nil
	}
	return base64.RawStdEncoding.DecodeString(clean)
}

func extFromMimeType(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return filepath.Ext(".png")
	}
}
