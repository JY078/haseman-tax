package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxConversionFileBytes = 20 << 20

func readConversionUpload(c *gin.Context, input *orderTrackingImportInput) (*model.OrderConversion, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 3*maxConversionFileBytes)
	if err := c.Request.ParseMultipartForm(maxConversionFileBytes); err != nil {
		return nil, fmt.Errorf("文件上传失败，每个文件最大 20 MB")
	}
	defer c.Request.MultipartForm.RemoveAll()
	if err := json.Unmarshal([]byte(c.PostForm("payload")), input); err != nil {
		return nil, fmt.Errorf("订单数据格式错误")
	}
	read := func(field string) (string, []byte, error) {
		header, err := c.FormFile(field)
		if err != nil {
			return "", nil, fmt.Errorf("缺少文件：%s", field)
		}
		file, err := header.Open()
		if err != nil {
			return "", nil, fmt.Errorf("无法读取文件")
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, maxConversionFileBytes+1))
		if err != nil || len(data) == 0 || len(data) > maxConversionFileBytes {
			return "", nil, fmt.Errorf("文件不能为空且不能超过 20 MB")
		}
		name := path.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
		return name, data, nil
	}
	record := &model.OrderConversion{}
	var err error
	record.SourceName, record.SourceData, err = read("source_file")
	if err != nil {
		return nil, err
	}
	record.MCFName, record.MCFData, err = read("mcf_file")
	if err != nil {
		return nil, err
	}
	return record, nil
}

func conversionFingerprint(record *model.OrderConversion, input orderTrackingImportInput) string {
	hash := sha256.New()
	// JSON encodes field boundaries; filenames are excluded so retries stay idempotent.
	json.NewEncoder(hash).Encode(struct {
		User  uint
		Input orderTrackingImportInput
	}{record.ImportedBy, input})
	sourceHash := sha256.Sum256(record.SourceData)
	mcfHash := sha256.Sum256(record.MCFData)
	hash.Write(sourceHash[:])
	hash.Write(mcfHash[:])
	return hex.EncodeToString(hash.Sum(nil))
}

func conversionQuery(user model.User) *gorm.DB {
	query := database.DB.Model(&model.OrderConversion{})
	if !canViewAllOrderTracking(user) {
		query = query.Where("platform IN ?", userOrderPlatforms(user))
	}
	return query
}

func ListOrderConversions(c *gin.Context) {
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	platform, allowed := accessibleOrderTrackingPlatform(user, c.Query("platform"))
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看该平台历史记录"})
		return
	}
	query := conversionQuery(user)
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	pageNumber, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if pageNumber < 1 {
		pageNumber = 1
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(500, gin.H{"error": "历史记录加载失败"})
		return
	}
	items := []model.OrderConversion{}
	if err := query.Select("id", "platform", "imported_by", "source_name", "mcf_name", "created_at").Order("id desc").Limit(20).Offset((pageNumber - 1) * 20).Find(&items).Error; err != nil {
		c.JSON(500, gin.H{"error": "历史记录加载失败"})
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func DownloadOrderConversion(c *gin.Context) {
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(400, gin.H{"error": "无效记录编号"})
		return
	}
	field := c.Param("file")
	if field != "source" && field != "mcf" {
		c.JSON(400, gin.H{"error": "无效文件类型"})
		return
	}
	var record model.OrderConversion
	err = conversionQuery(user).Select(field+"_name", field+"_data").First(&record, id).Error
	if err != nil {
		c.JSON(404, gin.H{"error": "文件不存在或无权访问"})
		return
	}
	name, data := record.SourceName, record.SourceData
	if field == "mcf" {
		name, data = record.MCFName, record.MCFData
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	c.Header("Cache-Control", "private, no-store")
	c.Data(200, "application/octet-stream", data)
}
