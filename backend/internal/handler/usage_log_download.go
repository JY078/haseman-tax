package handler

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func DownloadUsageLogImage(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid usage log id"})
		return
	}

	isAdmin := false
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err == nil {
		for _, role := range user.Roles {
			if role.Name == "admin" {
				isAdmin = true
				break
			}
		}
	}

	query := database.DB.Where("id = ? AND image_url != '' AND status = 'success'", id)
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	var logItem model.UsageLog
	if err := query.First(&logItem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query usage log"})
		return
	}

	filenameUser := user
	if isAdmin && logItem.UserID != userID {
		database.DB.Select("id", "username", "name", "platform", "department", "sub_department").First(&filenameUser, logItem.UserID)
	}
	filename := buildDownloadFilename(filenameUser, downloadFilenameParts{
		Brand:             c.Query("brand"),
		SKU:               c.Query("sku"),
		RequestedFilename: c.Query("filename"),
		SourcePath:        logItem.ImageURL,
	})
	annotate := c.Query("annotate") == "1"
	imageURL := strings.TrimSpace(logItem.ImageURL)
	if uploadsPath := extractUploadsPath(imageURL); uploadsPath != "" {
		sourcePath := filepath.Clean(strings.TrimPrefix(uploadsPath, "/"))
		localSourcePath, localErr := resolveLocalUploadPath(sourcePath)
		incrementUsageLogDownloadCount(logItem.ID)
		c.Header("Content-Disposition", contentDisposition(filename))
		if localErr == nil {
			if !annotate {
				c.File(localSourcePath)
				return
			}
			if err := writeDownloadedImage(c, localSourcePath, sourcePath, filename, true); err != nil {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "failed to annotate image metadata"})
			}
			return
		}
		if err := proxyDownloadStoredImage(c, uploadsPath, filename, annotate); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		}
		return
	}

	if err := proxyDownloadRemoteImage(c, imageURL, filename, annotate); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "download failed"})
		return
	}
	incrementUsageLogDownloadCount(logItem.ID)
}

func extractUploadsPath(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	if strings.HasPrefix(rawURL, "/uploads/") {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if strings.HasPrefix(parsed.Path, "/uploads/") {
		return parsed.Path
	}
	return ""
}

func proxyDownloadRemoteImage(c *gin.Context, rawURL string, filename string, annotate bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("invalid remote image url")
	}
	if isPrivateHost(parsed.Hostname()) {
		return fmt.Errorf("private remote image host is not allowed")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remote image responded %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", contentDisposition(filename))
	if resp.ContentLength > 0 {
		c.Header("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	}
	if !annotate {
		_, err = io.Copy(c.Writer, resp.Body)
		return err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	data, err = storage.EnsureSyntheticPerformerXMP(data, contentType)
	if err != nil {
		return err
	}
	c.Header("Content-Length", strconv.Itoa(len(data)))
	_, err = c.Writer.Write(data)
	return err
}

func isPrivateHost(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return true
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}
	return false
}

func incrementUsageLogDownloadCount(id uint) {
	database.DB.Model(&model.UsageLog{}).
		Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1))
}
