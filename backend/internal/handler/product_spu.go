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
	"gorm.io/gorm"
)

var productCertificationTypes = map[string]string{
	"cec":     "CEC",
	"doe":     "DOE",
	"cupc":    "cUPC",
	"nsf 61":  "NSF 61",
	"nsf 372": "NSF 372",
	"wars":    "WARS",
	"acs":     "ACS",
}

const productAttachmentOrder = "certification_date DESC, id DESC"

func AdminListProductSPUs(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	query := database.DB.Model(&model.ProductSPU{})
	query = filterProductSPUsByKeyword(query, keyword)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count product SPUs"})
		return
	}

	var items []model.ProductSPU
	if err := query.Preload("SKUs").Preload("Attachments", func(db *gorm.DB) *gorm.DB {
		return db.Order(productAttachmentOrder)
	}).Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SPUs"})
		return
	}

	for i := range items {
		items[i].SKUCount = len(items[i].SKUs)
	}
	hydrateProductSPUPlatformSKUs(items)
	hydrateProductAuditNames(items)

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func SearchProductSPUs(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	limit := 30

	var items []model.ProductSPU
	query := database.DB.Order("spu asc").Limit(limit).Preload("SKUs").Preload("Attachments", func(db *gorm.DB) *gorm.DB {
		return db.Order(productAttachmentOrder)
	})
	query = filterProductSPUsByKeyword(query, keyword)

	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SPUs"})
		return
	}
	hydrateProductAuditNames(items)
	hydrateProductSPUPlatformSKUs(items)
	c.JSON(http.StatusOK, items)
}

func filterProductSPUsByKeyword(query *gorm.DB, keyword string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return query
	}
	like := "%" + keyword + "%"
	return query.Where(`
		product_spus.spu ILIKE ? OR product_spus.name ILIKE ? OR product_spus.brand ILIKE ? OR
		product_spus.model ILIKE ? OR product_spus.material ILIKE ? OR product_spus.category1 ILIKE ? OR
		product_spus.category2 ILIKE ? OR EXISTS (
			SELECT 1 FROM product_skus
			WHERE product_skus.spu_id = product_spus.id
			AND (
				product_skus.sku ILIKE ? OR product_skus.asin ILIKE ? OR product_skus.name ILIKE ? OR
				product_skus.style_name ILIKE ? OR product_skus.color ILIKE ? OR product_skus.size ILIKE ? OR
				product_skus.feature ILIKE ? OR product_skus.material ILIKE ? OR product_skus.category1 ILIKE ? OR
				product_skus.category2 ILIKE ? OR product_skus.category3 ILIKE ?
			)
		)`, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like)
}

func AdminCreateProductSPU(c *gin.Context) {
	item, ok := bindProductSPUInput(c)
	if !ok {
		return
	}
	if productSPUExists(item.SPU, 0) {
		c.JSON(http.StatusConflict, gin.H{"error": "SPU 已存在，请使用其他编号"})
		return
	}
	item.UpdatedBy = currentUserID(c)
	if err := database.DB.Create(&item).Error; err != nil {
		if isDuplicateProductSPUError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "SPU 已存在，请使用其他编号"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product SPU: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func AdminUpdateProductSPU(c *gin.Context) {
	id, ok := parseProductSPUID(c)
	if !ok {
		return
	}

	var existing model.ProductSPU
	if err := database.DB.First(&existing, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product SPU not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SPU"})
		return
	}

	item, ok := bindProductSPUInput(c)
	if !ok {
		return
	}
	if item.SPU != existing.SPU {
		c.JSON(http.StatusConflict, gin.H{"error": "SPU 编号创建后不可修改"})
		return
	}
	existing.Name = item.Name
	existing.Brand = item.Brand
	existing.Model = item.Model
	existing.Material = item.Material
	existing.Category1 = item.Category1
	existing.Category2 = item.Category2
	existing.UpdatedBy = currentUserID(c)

	if err := database.DB.Save(&existing).Error; err != nil {
		if isDuplicateProductSPUError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "SPU 已存在，请使用其他编号"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product SPU: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func productSPUExists(spu string, excludeID uint) bool {
	query := database.DB.Model(&model.ProductSPU{}).Where("spu = ?", strings.TrimSpace(spu))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	return query.Count(&count).Error == nil && count > 0
}

func isDuplicateProductSPUError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "idx_product_spus_spu") || strings.Contains(message, "SQLSTATE 23505")
}

func AdminDeleteProductSPU(c *gin.Context) {
	id, ok := parseProductSPUID(c)
	if !ok {
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("spu_id = ?", id).Delete(&model.ProductSKU{}).Error; err != nil {
			return err
		}
		if err := tx.Where("spu_id = ?", id).Delete(&model.ProductSPUAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.ProductSPU{}, id).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product SPU"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product SPU deleted successfully"})
}

type batchDeleteProductSPUInput struct {
	IDs []uint `json:"ids"`
}

func AdminBatchDeleteProductSPUs(c *gin.Context) {
	var input batchDeleteProductSPUInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要删除的 SPU"})
		return
	}
	ids := uniqueProductSPUIDs(input.IDs)
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要删除的 SPU"})
		return
	}
	if len(ids) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次最多删除 100 个 SPU"})
		return
	}

	var deleted int64
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("spu_id IN ?", ids).Delete(&model.ProductSKU{}).Error; err != nil {
			return err
		}
		if err := tx.Where("spu_id IN ?", ids).Delete(&model.ProductSPUAttachment{}).Error; err != nil {
			return err
		}
		result := tx.Where("id IN ?", ids).Delete(&model.ProductSPU{})
		deleted = result.RowsAffected
		return result.Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "批量删除 SPU 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func uniqueProductSPUIDs(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	ids := make([]uint, 0, len(values))
	for _, id := range values {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func AdminGetProductSPU(c *gin.Context) {
	id, ok := parseProductSPUID(c)
	if !ok {
		return
	}

	var spu model.ProductSPU
	if err := database.DB.Preload("SKUs").Preload("Attachments", func(db *gorm.DB) *gorm.DB {
		return db.Order(productAttachmentOrder)
	}).First(&spu, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product SPU not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SPU"})
		return
	}
	spu.SKUCount = len(spu.SKUs)
	items := []model.ProductSPU{spu}
	hydrateProductSPUPlatformSKUs(items)
	hydrateProductAuditNames(items)
	spu = items[0]
	c.JSON(http.StatusOK, spu)
}

func hydrateProductSPUPlatformSKUs(items []model.ProductSPU) {
	allSKUs := make([]model.ProductSKU, 0)
	for _, item := range items {
		allSKUs = append(allSKUs, item.SKUs...)
	}
	hydrateProductPlatformSKUs(allSKUs)
	byID := make(map[uint]model.ProductSKU, len(allSKUs))
	for _, sku := range allSKUs {
		byID[sku.ID] = sku
	}
	for i := range items {
		for j := range items[i].SKUs {
			if sku, ok := byID[items[i].SKUs[j].ID]; ok {
				items[i].SKUs[j].PlatformSKUs = sku.PlatformSKUs
			}
		}
	}
}

func AdminUploadProductSPUModel(c *gin.Context) {
	id, ok := parseProductSPUID(c)
	if !ok {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	url, filename, fileType, err := uploadProductSPUAsset(c, "model", file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var spu model.ProductSPU
	if err := database.DB.First(&spu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product SPU not found"})
		return
	}

	spu.ModelURL = url
	spu.UpdatedBy = currentUserID(c)
	if err := database.DB.Save(&spu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存模型地址失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":       url,
		"file_name": filename,
		"file_type": fileType,
		"message":   "模型上传成功",
	})
}

func AdminUploadProductSPUAttachment(c *gin.Context) {
	id, ok := parseProductSPUID(c)
	if !ok {
		return
	}

	certificationType, typeOK := normalizeProductCertificationType(c.PostForm("certification_type"))
	certificationDate, dateOK := normalizeProductCertificationDate(c.PostForm("certification_date"))
	if !typeOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择有效的认证类型"})
		return
	}
	if !dateOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写有效的认证日期"})
		return
	}

	certificateFile, err := c.FormFile("certificate_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传认证信息文件"})
		return
	}
	testReportFile, err := c.FormFile("test_report_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传测试报告"})
		return
	}

	certificateURL, certificateName, certificateFileType, err := uploadProductSPUAsset(c, "attachment", certificateFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	testReportURL, testReportName, testReportType, err := uploadProductSPUAsset(c, "attachment", testReportFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	attachment := model.ProductSPUAttachment{
		SPUID:             id,
		Label:             certificationType,
		CertificationType: certificationType,
		CertificationDate: certificationDate,
		FileURL:           certificateURL,
		FileName:          certificateName,
		FileType:          certificateFileType,
		TestReportURL:     testReportURL,
		TestReportName:    testReportName,
		TestReportType:    testReportType,
		UpdatedBy:         currentUserID(c),
	}
	if err := database.DB.Create(&attachment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存认证资料失败"})
		return
	}
	touchProductSPU(id, attachment.UpdatedBy)

	c.JSON(http.StatusCreated, attachment)
}

func AdminUpdateProductSPUAttachment(c *gin.Context) {
	spuID, ok := parseProductSPUID(c)
	if !ok {
		return
	}
	attachmentID, ok := parseProductSPUAttachmentID(c)
	if !ok {
		return
	}

	var attachment model.ProductSPUAttachment
	if err := database.DB.Where("id = ? AND spu_id = ?", attachmentID, spuID).First(&attachment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attachment"})
		return
	}
	if attachment.CertificationType != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "认证记录不可覆盖，请新增一条记录"})
		return
	}

	var input struct {
		Label string `json:"label"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	attachment.Label = strings.TrimSpace(input.Label)
	if attachment.Label == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "认证标签不能为空"})
		return
	}
	attachment.UpdatedBy = currentUserID(c)

	if err := database.DB.Save(&attachment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新认证标签失败"})
		return
	}
	touchProductSPU(spuID, attachment.UpdatedBy)
	c.JSON(http.StatusOK, attachment)
}

func AdminListProductSPUAttachments(c *gin.Context) {
	spuID, ok := parseProductSPUID(c)
	if !ok {
		return
	}

	var items []model.ProductSPUAttachment
	if err := database.DB.Where("spu_id = ?", spuID).Order(productAttachmentOrder).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attachments"})
		return
	}
	hydrateProductAttachmentAuditNames(items)
	c.JSON(http.StatusOK, items)
}

func normalizeProductCertificationType(value string) (string, bool) {
	canonical, ok := productCertificationTypes[strings.ToLower(strings.TrimSpace(value))]
	return canonical, ok
}

func normalizeProductCertificationDate(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil || parsed.Format("2006-01-02") != trimmed {
		return "", false
	}
	return trimmed, true
}

func AdminDeleteProductSPUAttachment(c *gin.Context) {
	spuID, ok := parseProductSPUID(c)
	if !ok {
		return
	}
	attachmentID, ok := parseProductSPUAttachmentID(c)
	if !ok {
		return
	}

	if err := database.DB.Where("id = ? AND spu_id = ?", attachmentID, spuID).Delete(&model.ProductSPUAttachment{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除认证资料失败"})
		return
	}
	touchProductSPU(spuID, currentUserID(c))
	c.JSON(http.StatusOK, gin.H{"message": "认证资料已删除"})
}

func hydrateProductAuditNames(items []model.ProductSPU) {
	userIDs := make([]uint, 0, len(items))
	for i := range items {
		userIDs = append(userIDs, items[i].UpdatedBy)
		for j := range items[i].SKUs {
			userIDs = append(userIDs, items[i].SKUs[j].UpdatedBy)
		}
		for j := range items[i].Attachments {
			userIDs = append(userIDs, items[i].Attachments[j].UpdatedBy)
		}
	}
	names := loadUserDisplayNames(userIDs)
	for i := range items {
		items[i].UpdatedByName = names[items[i].UpdatedBy]
		for j := range items[i].SKUs {
			items[i].SKUs[j].UpdatedByName = names[items[i].SKUs[j].UpdatedBy]
		}
		for j := range items[i].Attachments {
			items[i].Attachments[j].UpdatedByName = names[items[i].Attachments[j].UpdatedBy]
		}
	}
}

func hydrateProductAttachmentAuditNames(items []model.ProductSPUAttachment) {
	userIDs := make([]uint, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.UpdatedBy)
	}
	names := loadUserDisplayNames(userIDs)
	for i := range items {
		items[i].UpdatedByName = names[items[i].UpdatedBy]
	}
}

func touchProductSPU(spuID uint, userID uint) {
	if spuID == 0 {
		return
	}
	database.DB.Model(&model.ProductSPU{}).Where("id = ?", spuID).Updates(map[string]interface{}{
		"updated_by": userID,
		"updated_at": time.Now(),
	})
}

func bindProductSPUInput(c *gin.Context) (model.ProductSPU, bool) {
	var input struct {
		SPU       string `json:"spu"`
		Name      string `json:"name"`
		Brand     string `json:"brand"`
		Model     string `json:"model"`
		Material  string `json:"material"`
		Category1 string `json:"category1"`
		Category2 string `json:"category2"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return model.ProductSPU{}, false
	}

	item := model.ProductSPU{
		SPU:       strings.TrimSpace(input.SPU),
		Name:      strings.TrimSpace(input.Name),
		Brand:     strings.TrimSpace(input.Brand),
		Model:     strings.TrimSpace(input.Model),
		Material:  strings.TrimSpace(input.Material),
		Category1: strings.TrimSpace(input.Category1),
		Category2: strings.TrimSpace(input.Category2),
	}
	if item.SPU == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SPU is required"})
		return model.ProductSPU{}, false
	}
	return item, true
}

func uploadProductSPUAsset(c *gin.Context, kind string, file *multipart.FileHeader) (string, string, string, error) {
	if file.Size <= 0 {
		return "", "", "", http.ErrMissingFile
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		if kind == "model" {
			ext = ".stp"
		} else {
			ext = ".pdf"
		}
	}

	safeName := strings.ReplaceAll(strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)), " ", "_")
	tmpDir := "./uploads/product-spu-files"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", "", "", err
	}

	tmpPath := filepath.Join(tmpDir, safeName+"_"+time.Now().Format("20060102150405.000000000")+ext)
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		return "", "", "", err
	}

	destDir := "products/product-spu"
	if kind == "attachment" {
		destDir = "products/product-spu-attachments"
	}

	url, err := storage.MoveToOSS(tmpPath, destDir)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", "", "", err
	}

	fileType := detectProductSPUFileType(ext)
	return url, file.Filename, fileType, nil
}

func detectProductSPUFileType(ext string) string {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return "image"
	case ".pdf":
		return "pdf"
	default:
		return "file"
	}
}

func detectProductSPUFileTypeFromURL(url string) string {
	return detectProductSPUFileType(filepath.Ext(url))
}

func parseProductSPUID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return 0, false
	}
	return uint(id), true
}

func parseProductSPUAttachmentID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("attachment_id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return 0, false
	}
	return uint(id), true
}
