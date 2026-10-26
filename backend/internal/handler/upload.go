package handler

import (
	"net/http"
	"strings"

	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

/** UploadImage 接收 base64 编码的图片，保存到服务器并返回可访问的 URL */
func UploadImage(c *gin.Context) {
	if strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "解析图片上传失败: " + err.Error()})
			return
		}
		if c.Request.MultipartForm != nil {
			defer c.Request.MultipartForm.RemoveAll()
		}
		fileHeader, err := c.FormFile("image")
		if err != nil {
			fileHeader, err = c.FormFile("file")
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "图片文件不能为空"})
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "读取图片文件失败"})
			return
		}
		defer file.Close()

		mimeType := c.PostForm("mime_type")
		if mimeType == "" {
			mimeType = fileHeader.Header.Get("Content-Type")
		}
		if mimeType == "" {
			mimeType = "image/png"
		}

		url, err := storage.SavePromptImageReader(file, mimeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"url": url})
		return
	}

	var input struct {
		Data     string `json:"data"`
		MimeType string `json:"mime_type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Data == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片数据不能为空"})
		return
	}

	// 去掉 data:image/xxx;base64, 前缀（如果有）
	data := input.Data
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}

	mimeType := input.MimeType
	if mimeType == "" {
		mimeType = "image/png"
	}

	url, err := storage.SavePromptImage(data, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
