package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/config"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

const maxImageMetadataBatchFiles = 100
const maxImageMetadataBatchMemory = 100 * 1024 * 1024

// AddSyntheticPerformerMetadataImage annotates one image in memory and
// returns it as an attachment. The source image is not persisted; the
// annotated download copy is also written to NAS when download sync is enabled.
func AddSyntheticPerformerMetadataImage(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxImageMetadataBatchMemory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read image"})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please select an image"})
		return
	}
	if file.Size > maxImageMetadataBatchMemory {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "image exceeds the 100 MB limit"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open image"})
		return
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read image"})
		return
	}
	mimeType := http.DetectContentType(data)
	if !isSupportedMetadataImage(data, mimeType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a supported PNG, JPEG, or WebP image"})
		return
	}
	data, err = storage.EnsureSyntheticPerformerXMP(data, mimeType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to process image"})
		return
	}
	user := getDownloadUser(c)
	filename := safeArchiveImageName(file.Filename, map[string]int{})
	if brand, sku := strings.TrimSpace(c.PostForm("brand")), strings.TrimSpace(c.PostForm("sku")); brand != "" && sku != "" {
		filename = buildDownloadFilename(user, downloadFilenameParts{
			Brand:             brand,
			SKU:               sku,
			RequestedFilename: c.PostForm("filename"),
			SourcePath:        file.Filename,
		})
	}
	if nasDir := strings.TrimSpace(config.GetEnv("DOWNLOAD_NAS_DIR", "")); nasDir != "" && shouldSaveDownloadToNAS(user) {
		targetDir := resolveDownloadNASDir(nasDir, user)
		if err := writeDownloadDataToNAS(data, targetDir, filename); err != nil {
			log.Printf("Warning: failed to copy annotated download to NAS: filename=%q err=%v", filename, err)
		}
	}
	c.Header("Content-Disposition", contentDisposition(filename))
	c.Data(http.StatusOK, mimeType, data)
}

// AddSyntheticPerformerMetadataBatch processes uploaded images in memory and
// returns a ZIP. Uploaded files are not saved to local storage, OSS, or DB.
func AddSyntheticPerformerMetadataBatch(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxImageMetadataBatchMemory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read uploaded images"})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	files := c.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please select at least one image"})
		return
	}
	if len(files) > maxImageMetadataBatchFiles {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("a batch supports at most %d images", maxImageMetadataBatchFiles)})
		return
	}

	archive, err := buildSyntheticPerformerArchive(files)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filename := "synthetic-performer-images-" + time.Now().Format("20060102-150405") + ".zip"
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", contentDisposition(filename))
	c.Header("Content-Length", strconv.Itoa(len(archive)))
	c.Data(http.StatusOK, "application/zip", archive)
}

// UploadDownloadArchiveToNAS stores a client-generated download archive in the
// same per-user NAS location used by server-side downloads.
func UploadDownloadArchiveToNAS(c *gin.Context) {
	user := getDownloadUser(c)
	nasDir := strings.TrimSpace(config.GetEnv("DOWNLOAD_NAS_DIR", ""))
	if nasDir == "" || !shouldSaveDownloadToNAS(user) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "NAS download sync is not configured"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please select an archive"})
		return
	}
	if file.Size <= 0 || file.Size > maxImageMetadataBatchMemory {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "archive exceeds the 100 MB limit"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open archive"})
		return
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read archive"})
		return
	}
	filename := filepath.Base(strings.TrimSpace(file.Filename))
	if filename == "." || filename == "" {
		filename = "workflow-images.zip"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".zip") {
		filename += ".zip"
	}
	if err := writeDownloadDataToNAS(data, resolveDownloadNASDir(nasDir, user), filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save archive to NAS"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"saved": true})
}

func buildSyntheticPerformerArchive(files []*multipart.FileHeader) ([]byte, error) {
	type processedImage struct {
		name string
		data []byte
	}
	images := make([]processedImage, 0, len(files))
	usedNames := make(map[string]int, len(files))

	for _, file := range files {
		name := safeArchiveImageName(file.Filename, usedNames)
		src, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s", name)
		}
		data, readErr := io.ReadAll(src)
		src.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read %s", name)
		}

		mimeType := http.DetectContentType(data)
		if !isSupportedMetadataImage(data, mimeType) {
			return nil, fmt.Errorf("%s is not a supported PNG, JPEG, or WebP image", name)
		}
		data, err = storage.EnsureSyntheticPerformerXMP(data, mimeType)
		if err != nil {
			return nil, fmt.Errorf("failed to process %s: %w", name, err)
		}
		images = append(images, processedImage{name: name, data: data})
	}

	var output bytes.Buffer
	zw := zip.NewWriter(&output)
	for _, image := range images {
		entry, err := zw.Create(image.name)
		if err != nil {
			return nil, fmt.Errorf("failed to create ZIP entry: %w", err)
		}
		if _, err := entry.Write(image.data); err != nil {
			return nil, fmt.Errorf("failed to write ZIP entry: %w", err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finish ZIP: %w", err)
	}
	return output.Bytes(), nil
}

func isSupportedMetadataImage(data []byte, mimeType string) bool {
	if strings.HasPrefix(mimeType, "image/png") {
		return len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	}
	if strings.HasPrefix(mimeType, "image/jpeg") {
		return len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8
	}
	return len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
}

func safeArchiveImageName(filename string, used map[string]int) string {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/"))
	if name == "" || name == "." {
		name = "image.png"
	}
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	candidate := name
	for index := 2; used[strings.ToLower(candidate)] > 0; index++ {
		candidate = fmt.Sprintf("%s-%d%s", base, index, ext)
	}
	used[strings.ToLower(candidate)] = 1
	return candidate
}
