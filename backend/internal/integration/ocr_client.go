package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OCRClient is a client for calling the Python OCR service.
type OCRClient struct {
	baseURL string
	client  *http.Client
}

// NewOCRClient creates a new OCR client with the specified base URL.
func NewOCRClient(baseURL string) *OCRClient {
	return &OCRClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// BBox represents a bounding box for OCR elements.
type BBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// LayerInfo contains information about a detected text layer.
type LayerInfo struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	BBox       BBox    `json:"bbox"`
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence"`
}

// OCRResult represents the result from the OCR analyze endpoint.
type OCRResult struct {
	Layers []LayerInfo `json:"layers"`
}

// ComposeLayer represents a layer to be composed into an image.
type ComposeLayer struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	BBox    BBox   `json:"bbox"`
}

// ComposeRequest represents the request to compose an image.
type ComposeRequest struct {
	ImageData string         `json:"image_data"`
	Layers    []ComposeLayer `json:"layers"`
}

// ComposeResponse represents the response from the compose endpoint.
type ComposeResponse struct {
	ImageData string `json:"image_data"`
}

// AnalyzeImage calls the OCR analyze endpoint with the given image data.
func (c *OCRClient) AnalyzeImage(ctx context.Context, imageData string) (*OCRResult, error) {
	reqBody := map[string]string{"image_data": imageData}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ocr/analyze", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OCR analyze: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OCR analyze returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result OCRResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OCR analyze response: %w", err)
	}

	return &result, nil
}

// ComposeImage calls the OCR compose endpoint with the given request.
func (c *OCRClient) ComposeImage(ctx context.Context, req *ComposeRequest) (*ComposeResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ocr/compose", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call OCR compose: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OCR compose returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result ComposeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OCR compose response: %w", err)
	}

	return &result, nil
}