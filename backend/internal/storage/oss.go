package storage

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSClient OSS 客户端单例
var OSSClient *oss.Client

// OSSBucket OSS bucket 单例
var OSSBucket *oss.Bucket

// OSSDomain 自定义域名（如果配置了）
var OSSDomain string

// InitOSS 初始化 OSS 客户端
func InitOSS() error {
	bucketName := os.Getenv("OSS_BUCKET")
	endpoint := os.Getenv("OSS_ENDPOINT")
	OSSDomain = os.Getenv("OSS_DOMAIN")

	if bucketName == "" || endpoint == "" {
		fmt.Println("Warning: OSS not configured, using local storage")
		return nil
	}

	// 使用阿里云 AccessKey 初始化
	accessKeyID := os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET")

	if accessKeyID == "" || accessKeySecret == "" {
		fmt.Println("Warning: OSS credentials not configured, using local storage")
		return nil
	}

	var err error
	// 设置超时（连接 30 秒，读写 300 秒）
	OSSClient, err = oss.New(endpoint, accessKeyID, accessKeySecret,
		oss.Timeout(30, 300), // 连接超时 30s，写入超时 300s
	)
	if err != nil {
		return fmt.Errorf("init OSS client: %w", err)
	}

	OSSBucket, err = OSSClient.Bucket(bucketName)
	if err != nil {
		return fmt.Errorf("get OSS bucket: %w", err)
	}

	fmt.Printf("OSS initialized: bucket=%s, endpoint=%s\n", bucketName, endpoint)
	return nil
}

// IsOSSEnabled 检查 OSS 是否已配置
func IsOSSEnabled() bool {
	return OSSBucket != nil
}

// UploadToOSS 上传本地文件到 OSS，返回 OSS URL
func UploadToOSS(localPath string, objectKey string) (string, error) {
	if OSSBucket == nil {
		return "", fmt.Errorf("OSS not initialized")
	}

	// 规范化 objectKey（移除反斜线）
	objectKey = filepath.ToSlash(objectKey)

	err := OSSBucket.PutObjectFromFile(objectKey, localPath)
	if err != nil {
		return "", fmt.Errorf("upload to OSS: %w", err)
	}

	return buildOSSURL(objectKey), nil
}

// DeleteFromOSS 从 OSS 删除文件
func DeleteFromOSS(objectKey string) error {
	if OSSBucket == nil {
		return fmt.Errorf("OSS not initialized")
	}

	return OSSBucket.DeleteObject(objectKey)
}

// GetObjectKeyFromURL 从 URL 中提取 OSS object key
func GetObjectKeyFromURL(url string) string {
	// 移除域名部分，提取 object key
	if OSSDomain != "" {
		url = strings.Replace(url, OSSDomain, "", 1)
	} else {
		// 尝试移除常见的 OSS URL 前缀
		bucket := os.Getenv("OSS_BUCKET")
		if bucket != "" {
			// https://bucket.oss-cn-hangzhou.aliyuncs.com/path -> path
			ossPrefix := "https://" + bucket + ".oss-"
			url = strings.Replace(url, ossPrefix, "", 1)
			// 移除 region 部分 (oss-cn-hangzhou.aliyuncs.com/)
			idx := strings.Index(url, "/")
			if idx != -1 {
				url = url[idx+1:]
			}
		}
	}

	return strings.TrimPrefix(url, "/")
}

// SaveBase64ToOSS 将 base64 图片直接上传到 OSS（不保存本地）
func SaveBase64ToOSS(b64Data string, mimeType string, dir string) (string, error) {
	if !IsOSSEnabled() {
		// 如果没有配置 OSS，回退到本地存储
		return saveBase64ToDir(b64Data, mimeType, dir)
	}

	// 解码 base64
	data, err := decodeBase64Data(b64Data)
	if err != nil {
		return "", err
	}

	// 生成文件名
	ext := getImageExtension(mimeType)
	objectKey := fmt.Sprintf("%s/%d_%d.%s", dir, time.Now().Unix(), time.Now().Nanosecond(), ext)
	objectKey = filepath.ToSlash(objectKey)

	// 直接使用 bytes.Reader 避免 string 转换开销
	t0 := time.Now()
	err = OSSBucket.PutObject(objectKey, bytes.NewReader(data))
	log.Printf("[存储] 上传到OSS完成 耗时=%v 大小=%d", time.Since(t0).Round(time.Millisecond), len(data))
	if err != nil {
		return "", fmt.Errorf("upload base64 to OSS: %w", err)
	}

	return buildOSSURL(objectKey), nil
}

// MoveToOSS 将本地文件上传到 OSS 并删除本地文件，返回 OSS URL
func MoveToOSS(localPath string, destDir string) (string, error) {
	if !IsOSSEnabled() {
		// OSS 未配置，返回本地路径
		return "/" + filepath.ToSlash(localPath), nil
	}

	// 生成 OSS object key
	filename := filepath.Base(localPath)
	timestamp := time.Now().UnixNano()
	objectKey := fmt.Sprintf("%s/%d_%s", destDir, timestamp, filename)
	objectKey = filepath.ToSlash(objectKey)

	// 上传到 OSS
	url, err := UploadToOSS(localPath, objectKey)
	if err != nil {
		return "", err
	}

	// 删除本地文件
	os.Remove(localPath)

	return url, nil
}

// MoveToOSSWithKey 将本地文件上传到 OSS，使用指定的对象键
func MoveToOSSWithKey(localPath string, objectKey string) (string, error) {
	if !IsOSSEnabled() {
		return "/" + filepath.ToSlash(localPath), nil
	}

	// 规范化 objectKey
	objectKey = filepath.ToSlash(objectKey)

	// 上传到 OSS
	url, err := UploadToOSS(localPath, objectKey)
	if err != nil {
		return "", err
	}

	// 删除本地文件
	os.Remove(localPath)

	return url, nil
}

func buildOSSURL(objectKey string) string {
	if OSSDomain != "" {
		return strings.TrimSuffix(OSSDomain, "/") + "/" + objectKey
	}
	bucket := os.Getenv("OSS_BUCKET")
	if bucket == "" {
		bucket = "unknown"
	}
	endpoint := os.Getenv("OSS_ENDPOINT")
	if endpoint == "" {
		endpoint = "oss-cn-hangzhou.aliyuncs.com"
	}
	// 从 endpoint 提取 region
	// oss-cn-hangzhou.aliyuncs.com -> oss-cn-hangzhou.aliyuncs.com
	return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, objectKey)
}

// decodeBase64Data 解码 base64 数据
// 自动处理 data URL 格式（如 "data:image/png;base64,xxxxx"），剥离前缀后解码
func decodeBase64Data(b64Data string) ([]byte, error) {
	// 剥离 data URL 前缀（如 "data:image/png;base64," 或 "data:image/jpeg;base64,"）
	if idx := strings.Index(b64Data, ","); idx != -1 {
		b64Data = b64Data[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(b64Data)
		if err != nil {
			return nil, fmt.Errorf("decode base64: %w", err)
		}
	}
	return data, nil
}

// FetchFromOSS 从 OSS 下载对象内容，返回 body reader 和 content type。
// 用于后端代理 OSS 图片给前端（解决跨域问题）。
func FetchFromOSS(objectKey string) (io.ReadCloser, string, error) {
	if OSSBucket == nil {
		return nil, "", fmt.Errorf("OSS not initialized")
	}

	// 安全检查：禁止路径穿越
	objectKey = filepath.ToSlash(objectKey)
	if strings.Contains(objectKey, "..") {
		return nil, "", fmt.Errorf("forbidden path")
	}

	// 先获取对象元数据以获取 content-type
	meta, err := OSSBucket.GetObjectDetailedMeta(objectKey)
	contentType := "application/octet-stream"
	if err == nil {
		if ct := meta.Get("Content-Type"); ct != "" {
			contentType = ct
		}
	}

	resp, err := OSSBucket.GetObject(objectKey)
	if err != nil {
		return nil, "", err
	}

	return resp, contentType, nil
}
