package handler

import (
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

type productSKUFileUploadResponse struct {
	URL string `json:"url"`
}

func AdminUploadProductSKUFile(c *gin.Context) {
	id, ok := parseProductSKUID(c)
	if !ok {
		return
	}

	field := strings.TrimSpace(c.PostForm("field"))
	if field != "stp_model" && field != "auth_doc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid field"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	url, err := uploadProductSKUFile(c, field, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var sku model.ProductSKU
	if err := database.DB.First(&sku, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product SKU not found"})
		return
	}

	switch field {
	case "stp_model":
		sku.StpModelURL = url
	case "auth_doc":
		sku.AuthDocURL = url
	}

	if err := database.DB.Save(&sku).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件地址失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

func uploadProductSKUFile(c *gin.Context, field string, file *multipart.FileHeader) (string, error) {
	if file.Size <= 0 {
		return "", http.ErrMissingFile
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		switch field {
		case "stp_model":
			ext = ".stp"
		default:
			ext = ".pdf"
		}
	}

	tmpDir := "./uploads/product-sku-files"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	tmpPath := filepath.Join(tmpDir, strings.ReplaceAll(strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)), " ", "_")+"_"+time.Now().Format("20060102150405.000000000")+ext)
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		return "", err
	}

	destDir := "products/product-sku"
	url, err := storage.MoveToOSS(tmpPath, destDir)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return url, nil
}

func parseProductSKUID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return 0, false
	}
	return uint(id), true
}
