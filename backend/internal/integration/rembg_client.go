package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const RembgModel = "birefnet-general"

var ErrRembgUnavailable = errors.New("抠图服务暂未就绪，请稍后重试")

func (c *RembgClient) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return ErrRembgUnavailable
	}
	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("[rembg] health: %v", err)
		return ErrRembgUnavailable
	}
	defer resp.Body.Close()
	var health struct {
		Model  string `json:"model"`
		Status string `json:"status"`
	}
	if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&health) != nil || health.Model != RembgModel || health.Status != "ok" {
		log.Printf("[rembg] health mismatch: url=%s status=%d model=%q health=%q", c.baseURL, resp.StatusCode, health.Model, health.Status)
		return ErrRembgUnavailable
	}
	return nil
}

// Serialize CPU work across all background-removal entry points in this backend.
var rembgSlot = make(chan struct{}, 1)

type RembgClient struct {
	baseURL string
	client  *http.Client
}

func NewRembgClient(baseURL string) *RembgClient {
	return &RembgClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Minute},
	}
}

func (c *RembgClient) Remove(ctx context.Context, imageData, background string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	select {
	case rembgSlot <- struct{}{}:
		defer func() { <-rembgSlot }()
	case <-ctx.Done():
		return nil, fmt.Errorf("等待抠图超时: %w", ctx.Err())
	}
	body, err := json.Marshal(map[string]string{"image_data": imageData, "background": background})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/remove", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("[rembg] request: %v", err)
		return nil, ErrRembgUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		log.Printf("[rembg] status=%d body=%s", resp.StatusCode, data)
		var body struct {
			Detail string `json:"detail"`
		}
		if (resp.StatusCode == 400 || resp.StatusCode == 413 || resp.StatusCode == 503) && json.Unmarshal(data, &body) == nil && body.Detail != "" {
			return nil, errors.New(body.Detail)
		}
		return nil, errors.New("抠图未完成，请稍后重试")
	}
	const maxOutput = 100 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOutput+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxOutput {
		return nil, fmt.Errorf("抠图结果超过 100MB")
	}
	if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("抠图服务未返回有效 PNG: %w", err)
	}
	return data, nil
}
