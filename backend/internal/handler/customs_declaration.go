package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const customsMaxFile = 20 << 20

func customsPythonExecutable() string {
	for _, key := range []string{"CUSTOMS_PYTHON", "STP_WORKER_PYTHON"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
		if value := strings.TrimSpace(viper.GetString(key)); value != "" {
			return value
		}
	}
	return "python3"
}

func recognizeCustomsPDF(ctx context.Context, content []byte) (model.CustomsDeclaration, error) {
	var record model.CustomsDeclaration
	file, err := os.CreateTemp("", "customs-*.pdf")
	if err != nil {
		return record, err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(content); err != nil {
		file.Close()
		return record, err
	}
	if err = file.Close(); err != nil {
		return record, err
	}
	python := customsPythonExecutable()
	script := filepath.Join("internal", "customs", "parse.py")
	if _, err := os.Stat(script); err != nil {
		log.Printf("[Customs] Parser script unavailable: %v", err)
		return record, errors.New("报关单识别脚本不存在，请部署 internal/customs/parse.py 并检查后端工作目录")
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, script, file.Name())
	output, err := cmd.Output()
	if ctx.Err() != nil {
		return record, errors.New("识别超时，请减少 PDF 页数后重试")
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Log dependency failures without logging extracted document content.
			if bytes.Contains(exitErr.Stderr, []byte("ModuleNotFoundError")) {
				log.Printf("[Customs] Python dependency missing; executable=%q", python)
				return record, errors.New("报关单 Python 环境缺少依赖，请使用配置的 Python 执行 -m pip install -r internal/customs/requirements.txt")
			}
		} else {
			log.Printf("[Customs] Cannot start Python %q: %v", python, err)
			return record, errors.New("无法启动报关单 Python，请检查 CUSTOMS_PYTHON 或 STP_WORKER_PYTHON 的路径和执行权限")
		}
		var failure struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(output, &failure) == nil && failure.Error != "" {
			return record, fmt.Errorf("PDF 识别失败：%s", failure.Error)
		}
		return record, errors.New("PDF 识别服务不可用或文件已损坏，请检查 Python、pdfplumber 和 OCR 依赖")
	}
	if err = json.Unmarshal(output, &record); err != nil {
		return record, errors.New("PDF 识别结果无效")
	}
	if len(record.DeclarationNumber) != 18 || record.Fields == nil {
		return record, errors.New("未识别到有效海关编号")
	}
	return record, nil
}

func UploadCustomsDeclaration(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, customsMaxFile+(1<<20))
	if err := c.Request.ParseMultipartForm(customsMaxFile); err != nil {
		c.JSON(400, gin.H{"error": "请上传不超过 20 MB 的 PDF"})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	file, err := c.FormFile("file")
	if err != nil || file.Size > customsMaxFile || !strings.EqualFold(filepath.Ext(file.Filename), ".pdf") {
		c.JSON(400, gin.H{"error": "请上传不超过 20 MB 的 PDF"})
		return
	}
	stream, err := file.Open()
	if err != nil {
		c.JSON(400, gin.H{"error": "文件读取失败"})
		return
	}
	defer stream.Close()
	content, err := io.ReadAll(io.LimitReader(stream, customsMaxFile+1))
	if err != nil || len(content) > customsMaxFile || !bytes.HasPrefix(content, []byte("%PDF-")) {
		c.JSON(400, gin.H{"error": "文件不是有效的 PDF"})
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	var existing model.CustomsDeclaration
	err = database.DB.Select("id", "declaration_number").Where("file_hash = ?", hash).First(&existing).Error
	if err == nil {
		c.JSON(200, gin.H{"duplicate": true, "id": existing.ID, "declaration_number": existing.DeclarationNumber, "message": fmt.Sprintf("海关编号 %s 已导入，已跳过重复单据", existing.DeclarationNumber)})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(500, gin.H{"error": "查询报关单失败"})
		return
	}
	record, err := recognizeCustomsPDF(c.Request.Context(), content)
	if err != nil {
		c.JSON(422, gin.H{"error": err.Error()})
		return
	}
	err = database.DB.Select("id", "declaration_number").Where("declaration_number = ?", record.DeclarationNumber).First(&existing).Error
	if err == nil {
		c.JSON(200, gin.H{"duplicate": true, "id": existing.ID, "declaration_number": existing.DeclarationNumber, "message": fmt.Sprintf("海关编号 %s 已导入，已跳过重复单据", existing.DeclarationNumber)})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(500, gin.H{"error": "查询报关单失败"})
		return
	}
	record.FileName = filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/"))
	if len([]rune(record.FileName)) > 255 {
		c.JSON(400, gin.H{"error": "文件名过长"})
		return
	}
	record.FileHash, record.FileContent, record.UserID = hash, content, c.GetUint("userID")
	result := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "保存报关单失败"})
		return
	}
	if result.RowsAffected == 0 {
		if err := database.DB.Select("id", "declaration_number").Where("file_hash = ? OR declaration_number = ?", hash, record.DeclarationNumber).First(&existing).Error; err != nil {
			c.JSON(500, gin.H{"error": "查询已存在报关单失败"})
			return
		}
		c.JSON(200, gin.H{"duplicate": true, "id": existing.ID, "declaration_number": existing.DeclarationNumber, "message": fmt.Sprintf("海关编号 %s 已导入，已跳过重复单据", existing.DeclarationNumber)})
		return
	}
	c.JSON(201, gin.H{"id": record.ID, "message": "识别完成，报关单已保存", "warnings": record.Warnings})
}

func ListCustomsDeclarations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size < 1 || size > 100 {
		size = 20
	}
	query, ok := customsFilteredQuery(c)
	if !ok {
		return
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(500, gin.H{"error": "报关单查询失败"})
		return
	}
	records := []model.CustomsDeclaration{}
	if err := query.Omit("file_content", "raw_text").Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&records).Error; err != nil {
		c.JSON(500, gin.H{"error": "报关单查询失败"})
		return
	}
	c.JSON(200, gin.H{"items": records, "total": total, "page": page, "page_size": size})
}

func ExportCustomsDeclarations(c *gin.Context) {
	query, ok := customsFilteredQuery(c)
	if !ok {
		return
	}
	records := []model.CustomsDeclaration{}
	if err := query.Omit("file_content", "raw_text").Order("id DESC").Find(&records).Error; err != nil {
		c.JSON(500, gin.H{"error": "报关单报表导出失败"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"items": records, "total": len(records)})
}

func getCustomsRecord(c *gin.Context, includeFile bool) (*model.CustomsDeclaration, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(400, gin.H{"error": "无效报关单编号"})
		return nil, false
	}
	query := database.DB
	if includeFile {
		query = query.Select("id", "file_name", "file_content")
	} else {
		query = query.Omit("file_content")
	}
	var record model.CustomsDeclaration
	err = query.First(&record, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, gin.H{"error": "报关单不存在"})
		return nil, false
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "报关单读取失败"})
		return nil, false
	}
	return &record, true
}

func GetCustomsDeclaration(c *gin.Context) {
	if record, ok := getCustomsRecord(c, false); ok {
		c.JSON(200, record)
	}
}

func DownloadCustomsDeclaration(c *gin.Context) {
	if record, ok := getCustomsRecord(c, true); ok {
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": record.FileName}))
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(http.StatusOK, "application/pdf", record.FileContent)
	}
}
