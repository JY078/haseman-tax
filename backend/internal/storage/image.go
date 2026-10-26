package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

const (
	/** AI 生成的临时图片目录，会被定期清理 */
	UploadDir = "uploads/images"

	VideoUploadDir = "uploads/videos"
	AudioUploadDir = "uploads/audios"

	/** 手动上传的永久图片目录（提示词效果图等），不参与清理 */
	promptUploadDir = "uploads/prompts"

	/** 腾讯 AIGC 参考图临时目录：仅保留 10 分钟，由调用方负责删除 */
	tempRefDir = "uploads/temp_refs"

	ImageTTLDays = 90

	/** TempRefTTL 腾讯 AIGC 参考图存活时间 */
	TempRefTTL = 10 * time.Minute
)

/** saveBase64ToDir 将 base64 图片解码后保存到指定目录，返回可访问的 URL 路径 */
func saveBase64ToDir(b64Data string, mimeType string, dir string) (string, error) {
	if b64Data == "" {
		return "", fmt.Errorf("empty image data")
	}

	ext := "png"
	switch mimeType {
	case "image/jpeg", "image/jpg":
		ext = "jpg"
	case "image/webp":
		ext = "webp"
	case "image/gif":
		ext = "gif"
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(b64Data)
		if err != nil {
			return "", fmt.Errorf("decode base64: %w", err)
		}
	}

	filename := fmt.Sprintf("%d_%d.%s", time.Now().Unix(), time.Now().Nanosecond(), ext)
	filePath := filepath.Join(dir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("write image file: %w", err)
	}

	return "/" + filepath.ToSlash(filePath), nil
}

/** SaveBase64Image 保存 AI 生成的临时图片到 OSS 或本地，返回可访问的 URL */
func SaveBase64Image(b64Data string, mimeType string) (string, error) {
	return SaveBase64ToOSS(b64Data, mimeType, UploadDir)
}

/** SavePromptImage 保存手动上传的永久图片（如提示词效果图）到 OSS，不参与清理 */
func SavePromptImage(b64Data string, mimeType string) (string, error) {
	return SaveBase64ToOSS(b64Data, mimeType, uploadDirForMime(mimeType, promptUploadDir))
}

/** SaveTempRefImage 将 base64 参考图上传到 OSS 或本地。
 * 腾讯 AIGC 等需要 URL 的接口使用，文件在 TempRefTTL（10分钟）后应删除。
 * 调用方应通过 DeleteTempRefFiles 或 CleanExpiredTempRefs 清理文件。
 */
func SavePromptImageReader(reader io.Reader, mimeType string) (string, error) {
	return UploadImageReaderToOSS(reader, mimeType, uploadDirForMime(mimeType, promptUploadDir))
}

func uploadDirForMime(mimeType string, imageDir string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "video/") {
		return VideoUploadDir
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "audio/") {
		return AudioUploadDir
	}
	return imageDir
}

func SaveTempRefImage(b64Data string, mimeType string) (string, error) {
	return SaveBase64ToOSS(b64Data, mimeType, tempRefDir)
}

/** DeleteTempRefFiles 立即删除指定的临时参考图文件（用于请求完成后的精确清理）*/
func DeleteTempRefFiles(paths []string) {
	for _, p := range paths {
		// 只允许删除 temp_refs 目录下的文件，防止路径穿越
		clean := filepath.ToSlash(filepath.Clean(p))
		if strings.Contains(clean, tempRefDir) {
			_ = os.Remove(p)
		}
	}
}

/** CleanExpiredTempRefs 清理 uploads/temp_refs 下超过 TempRefTTL 的过期文件 */
func CleanExpiredTempRefs() error {
	entries, err := os.ReadDir(tempRefDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read temp_refs dir: %w", err)
	}
	cutoff := time.Now().Add(-TempRefTTL)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(tempRefDir, entry.Name()))
		}
	}
	return nil
}

/** SaveImageFromURL 从公开 URL 下载图片，上传到 OSS（或保存本地），返回 URL。
 * 适用于腾讯 AIGC 等异步接口返回 CDN URL 的场景。
 * 优化：直接下载到内存然后上传到 OSS，不保存本地临时文件。
 */
func SaveImageFromURL(imageURL string) (url string, mimeType string, err error) {
	// 直接下载到内存（不写入本地临时文件）
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download image: HTTP %d", resp.StatusCode)
	}

	mime := resp.Header.Get("Content-Type")
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	if mime == "" {
		mime = "image/png"
	}

	// 上传到 OSS
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read image body: %w", err)
	}

	url, err = UploadImageDataToOSSWithTimeout(data, mime, UploadDir, 180*time.Second)
	if err != nil {
		return "", "", fmt.Errorf("upload to OSS: %w", err)
	}

	return url, mime, nil
}

func SaveVideoFromURL(videoURL string) (url string, mimeType string, err error) {
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Get(videoURL)
	if err != nil {
		return "", "", fmt.Errorf("download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download video: HTTP %d", resp.StatusCode)
	}

	mime := resp.Header.Get("Content-Type")
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	if mime == "" || mime == "application/octet-stream" {
		mime = "video/mp4"
	}
	if strings.HasPrefix(mime, "image/") {
		return "", "", fmt.Errorf("download video: got image content type %s", mime)
	}
	if !strings.HasPrefix(mime, "video/") && mime != "application/mp4" {
		return "", "", fmt.Errorf("download video: unsupported content type %s", mime)
	}

	url, err = UploadImageReaderToOSSWithTimeout(resp.Body, mime, VideoUploadDir, 5*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("upload video to OSS: %w", err)
	}

	return url, mime, nil
}

// DownloadImageToMemory 从 URL 下载图片到内存
func DownloadImageToMemory(imageURL string) (data []byte, mimeType string, err error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return nil, "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download image: HTTP %d", resp.StatusCode)
	}

	mimeType = resp.Header.Get("Content-Type")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if mimeType == "" {
		mimeType = "image/png"
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read image body: %w", err)
	}

	return data, mimeType, nil
}

// UploadImageDataToOSS 将图片数据直接上传到 OSS
func UploadImageDataToOSS(data []byte, mimeType string, dir string) (string, error) {
	return UploadImageDataToOSSWithTimeout(data, mimeType, dir, 180*time.Second)
}

// UploadImageDataToOSSWithTimeout 将图片数据直接上传到 OSS，带超时控制
func UploadImageReaderToOSS(reader io.Reader, mimeType string, dir string) (string, error) {
	return UploadImageReaderToOSSWithTimeout(reader, mimeType, dir, 180*time.Second)
}

func UploadImageReaderToOSSWithTimeout(reader io.Reader, mimeType string, dir string, timeout time.Duration) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read upload body: %w", err)
	}
	return UploadImageDataToOSSWithTimeout(data, mimeType, dir, timeout)
}

func UploadImageDataToOSSWithTimeout(data []byte, mimeType string, dir string, timeout time.Duration) (string, error) {
	if !IsOSSEnabled() {
		// OSS 未配置，回退到本地存储
		return saveDataToDir(data, mimeType, dir)
	}

	ext := getImageExtension(mimeType)
	objectKey := fmt.Sprintf("%s/%d_%d.%s", dir, time.Now().Unix(), time.Now().Nanosecond(), ext)
	objectKey = filepath.ToSlash(objectKey)

	// 创建带超时控制的 context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	t0 := time.Now()
	// 使用带超时的 context 上传
	body := bytes.NewReader(data)
	done := make(chan error, 1)
	go func() {
		done <- OSSBucket.PutObject(objectKey, body)
	}()
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("upload to OSS timeout after %v", timeout)
	case err := <-done:
		log.Printf("[存储] 上传到OSS完成 耗时=%v 大小=%d", time.Since(t0).Round(time.Millisecond), len(data))
		if err != nil {
			return "", fmt.Errorf("upload to OSS: %w", err)
		}
		return buildOSSURL(objectKey), nil
	}
}

// saveDataToDir 将图片数据保存到本地目录
func saveDataToDir(data []byte, mimeType string, dir string) (string, error) {
	ext := getImageExtension(mimeType)
	filename := fmt.Sprintf("%d_%d.%s", time.Now().Unix(), time.Now().Nanosecond(), ext)
	filePath := filepath.Join(dir, filename)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return "/" + filepath.ToSlash(filePath), nil
}

// DownloadToTemp 下载图片到本地临时目录（带超时控制）
func saveReaderToDir(reader io.Reader, mimeType string, dir string) (string, error) {
	ext := getImageExtension(mimeType)
	filename := fmt.Sprintf("%d_%d.%s", time.Now().Unix(), time.Now().Nanosecond(), ext)
	filePath := filepath.Join(dir, filename)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return "/" + filepath.ToSlash(filePath), nil
}

func DownloadToTemp(imageURL string) (localPath string, mimeType string, err error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download image: HTTP %d", resp.StatusCode)
	}

	mimeType = resp.Header.Get("Content-Type")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if mimeType == "" {
		mimeType = "image/png"
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read image body: %w", err)
	}

	// 保存到临时目录
	ext := getImageExtension(mimeType)
	if err := os.MkdirAll(tempRefDir, 0755); err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}
	tmpPath := filepath.Join(tempRefDir, fmt.Sprintf("%d_%d.%s", time.Now().Unix(), time.Now().Nanosecond(), ext))
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return "", "", fmt.Errorf("write temp file: %w", err)
	}

	return tmpPath, mimeType, nil
}

// getImageExtension 获取图片扩展名
func getImageExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "video/mp4", "application/mp4":
		return "mp4"
	case "video/webm":
		return "webm"
	case "video/quicktime":
		return "mov"
	case "video/x-msvideo":
		return "avi"
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/wav", "audio/x-wav":
		return "wav"
	case "audio/mp4", "audio/x-m4a":
		return "m4a"
	default:
		return "png"
	}
}

/** CleanExpiredImages 清理过期图片。
 * 本地模式：清理 uploads/images 下超过 TTL 天的临时文件
 * OSS 模式：清理 OSS 上超过 TTL 天的临时文件（根据对象名中的时间戳判断）
 */
func CleanExpiredImages() error {
	// 1. 清理本地文件（如果有残留）
	cleanLocalExpiredImages()

	// 2. 清理 OSS 文件（如果 OSS 已启用）
	if IsOSSEnabled() {
		cleanOSSExpiredImages()
	}

	return nil
}

// cleanLocalExpiredImages 清理本地上传目录中过期的图片
func cleanLocalExpiredImages() {
	entries, err := os.ReadDir(UploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Read upload dir error: %v", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -ImageTTLDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(UploadDir, entry.Name())
			if err := os.Remove(path); err == nil {
				log.Printf("Cleaned local expired image: %s", path)
			}
		}
	}
}

// cleanOSSExpiredImages 清理 OSS 上超过 TTL 天的过期图片
// 对象名格式：uploads/images/1234567890_123.png
// 时间戳从文件名中提取（格式：Unix时间戳_纳秒.扩展名）
func cleanOSSExpiredImages() {
	if OSSBucket == nil {
		return
	}

	prefix := UploadDir + "/"
	marker := ""
	count := 0

	for {
		lsRes, err := OSSBucket.ListObjects(oss.Marker(marker), oss.Prefix(prefix), oss.MaxKeys(1000))
		if err != nil {
			log.Printf("OSS list objects error: %v", err)
			return
		}

		cutoff := time.Now().AddDate(0, 0, -ImageTTLDays)

		for _, object := range lsRes.Objects {
			// 从对象名中提取时间戳
			// 格式：uploads/images/1234567890_123456789.png
			timestamp, err := parseTimestampFromObjectKey(object.Key)
			if err != nil {
				// 无法解析时间戳，跳过
				continue
			}

			objectTime := time.Unix(timestamp, 0)
			if objectTime.Before(cutoff) {
				if err := OSSBucket.DeleteObject(object.Key); err == nil {
					log.Printf("Cleaned OSS expired image: %s", object.Key)
					count++
				} else {
					log.Printf("Failed to delete OSS object %s: %v", object.Key, err)
				}
			}
		}

		if !lsRes.IsTruncated {
			break
		}
		marker = lsRes.NextMarker
	}

	if count > 0 {
		log.Printf("OSS cleanup: removed %d expired images", count)
	}
}

// parseTimestampFromObjectKey 从对象名中解析 Unix 时间戳
// 格式：uploads/images/1234567890_123456789.png
// 返回：Unix 时间戳（秒）
func parseTimestampFromObjectKey(objectKey string) (int64, error) {
	// 获取文件名部分
	filename := filepath.Base(objectKey)
	// 移除扩展名
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	// 格式：1234567890_123456789
	// 时间戳是下划线前的部分
	parts := strings.Split(nameWithoutExt, "_")
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid object key format: %s", objectKey)
	}

	var timestamp int64
	for _, part := range parts {
		timestamp, _ = strconv.ParseInt(part, 10, 64)
		if timestamp > 1000000000 && timestamp < 2000000000 { // 合理的时间戳范围 (2001-2033)
			return timestamp, nil
		}
	}

	// 如果没找到合适的时间戳，尝试第一个部分
	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp from: %s", objectKey)
	}
	return timestamp, nil
}
