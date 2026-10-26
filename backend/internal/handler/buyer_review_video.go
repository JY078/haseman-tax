package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxReviewVideoSize = 100 << 20

func reviewVideoTypeAllowed(name, mime string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4":
		return mime == "video/mp4"
	case ".mov":
		return mime == "video/quicktime"
	case ".webm":
		return mime == "video/webm"
	}
	return false
}

func UploadBuyerReviewVideo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(400, gin.H{"error": "评论编号无效"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxReviewVideoSize+(1<<20))
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			c.JSON(413, gin.H{"error": "视频不能超过 100 MB"})
		} else {
			c.JSON(400, gin.H{"error": "视频上传格式无效"})
		}
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	file, err := c.FormFile("video")
	if err != nil || file.Size == 0 {
		c.JSON(400, gin.H{"error": "请选择视频文件"})
		return
	}
	if file.Size > maxReviewVideoSize {
		c.JSON(413, gin.H{"error": "视频不能超过 100 MB"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(400, gin.H{"error": "视频读取失败"})
		return
	}
	defer src.Close()
	detected, err := mimetype.DetectReader(src)
	if err != nil || !reviewVideoTypeAllowed(file.Filename, detected.String()) {
		c.JSON(400, gin.H{"error": "仅支持 MP4、MOV、WebM 视频，请检查文件内容"})
		return
	}
	if _, err = src.Seek(0, io.SeekStart); err != nil {
		c.JSON(400, gin.H{"error": "视频读取失败"})
		return
	}
	var review model.BuyerReview
	if err := database.DB.Select("id").First(&review, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "评论不存在"})
		} else {
			c.JSON(500, gin.H{"error": "评论读取失败"})
		}
		return
	}
	dir := filepath.Join("uploads", "review-videos")
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(500, gin.H{"error": "视频存储不可用"})
		return
	}
	path := filepath.Join(dir, uuid.NewString()+strings.ToLower(filepath.Ext(file.Filename)))
	keepLocal := false
	defer func() {
		if !keepLocal {
			_ = os.Remove(path)
		}
	}()
	dst, err := os.Create(path)
	if err != nil {
		c.JSON(500, gin.H{"error": "视频保存失败"})
		return
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		c.JSON(500, gin.H{"error": "视频保存失败"})
		return
	}
	objectKey := filepath.ToSlash(path)
	url := "/" + objectKey
	if storage.IsOSSEnabled() {
		url, err = storage.UploadToOSS(path, objectKey)
		if err != nil {
			c.JSON(500, gin.H{"error": "视频上传失败，请重试"})
			return
		}
	}
	result := database.DB.Model(&model.BuyerReview{}).Where("id = ?", id).Updates(map[string]any{"video_upload": url, "video": true})
	if result.Error != nil || result.RowsAffected == 0 {
		if storage.IsOSSEnabled() {
			_ = storage.DeleteFromOSS(objectKey)
		}
		c.JSON(500, gin.H{"error": "视频关联失败，请重试"})
		return
	}
	keepLocal = !storage.IsOSSEnabled()
	c.JSON(200, gin.H{"video_upload": url, "video": true})
}
