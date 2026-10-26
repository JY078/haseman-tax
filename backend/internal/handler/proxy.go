package handler

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

const maxProxySize = 50 * 1024 * 1024 // 50 MB, aligned with frontend reference image limit

// ProxyImage fetches images from OSS or public URLs and returns the content.
// Used by the frontend ImageEditModal to load images without CORS issues.
func ProxyImage(c *gin.Context) {
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	// Validate URL scheme
	parsedURL, err := url.Parse(targetURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	// If OSS is enabled and this is an OSS URL, fetch via OSS SDK (avoids HTTP round-trip).
	// URLs with image processing params must go through HTTP so OSS can apply resizing.
	if storage.IsOSSEnabled() && parsedURL.Query().Get("x-oss-process") == "" {
		objectKey := storage.GetObjectKeyFromURL(targetURL)
		if objectKey != "" && !strings.Contains(objectKey, "..") {
			body, contentType, err := storage.FetchFromOSS(objectKey)
			if err == nil && body != nil {
				defer body.Close()
				// Limit response size
				limited := io.LimitReader(body, maxProxySize+1)
				data, err := io.ReadAll(limited)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "read oss body failed"})
					return
				}
				if len(data) > maxProxySize {
					c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large"})
					return
				}
				c.Header("Content-Type", contentType)
				c.Header("Cache-Control", "public, max-age=86400")
				c.Data(http.StatusOK, contentType, data)
				return
			}
		}
	}

	// Fallback: direct HTTP fetch with size limit
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}
	req.Header.Set("User-Agent", "Haseman/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fetch failed"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": http.StatusText(resp.StatusCode)})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Limit response size
	limited := io.LimitReader(resp.Body, maxProxySize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read body failed"})
		return
	}
	if len(data) > maxProxySize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large"})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, resp.Header.Get("Content-Type"), data)
}
