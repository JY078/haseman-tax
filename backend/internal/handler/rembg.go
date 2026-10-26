package handler

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"advertising-image/backend/internal/integration"
	"advertising-image/backend/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Bound retained base64 payloads as well as inference concurrency.
var rembgTasks = make(chan struct{}, 3)

func acquireRembgTask(c *gin.Context) bool {
	select {
	case rembgTasks <- struct{}{}:
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 70<<20)
		return true
	default:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "抠图队列已满，请等待现有任务完成后重试"})
		return false
	}
}

func rembgClient() *integration.RembgClient {
	baseURL := strings.TrimSpace(viper.GetString("REMBG_SERVICE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8011"
	}
	return integration.NewRembgClient(baseURL)
}

func removeBackgroundImage(ctx context.Context, imageData, background string) (string, error) {
	data, err := rembgClient().Remove(ctx, imageData, background)
	if err != nil {
		return "", err
	}
	return storage.SaveBase64Image(base64.StdEncoding.EncodeToString(data), "image/png")
}
