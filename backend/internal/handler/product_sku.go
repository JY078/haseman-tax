package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var downloadBrands = []string{
	"DEAROCEAN",
	"FEECOOL",
	"FORIOUS",
	"QFN",
	"SUNOUTLY",
	"\u65e0\u54c1\u724c",
	"Androme",
	"ANZA",
	"FURUISI",
	"VEUUI",
}

func GetDownloadOptions(c *gin.Context) {
	user := getDownloadUser(c)
	c.JSON(http.StatusOK, gin.H{
		"brands":         downloadBrands,
		"platform":       user.Platform,
		"department":     user.Department,
		"sub_department": user.SubDepartment,
	})
}

func isDownloadBrandAllowed(value string) bool {
	value = strings.TrimSpace(value)
	for _, brand := range downloadBrands {
		if value == brand {
			return true
		}
	}
	return false
}

func SearchProductSKUs(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	limit := 30

	var items []model.ProductSKU
	query := database.DB.Where(`EXISTS (
		SELECT 1 FROM product_spus
		WHERE product_spus.id = product_skus.spu_id OR product_spus.spu = product_skus.spu
	)`).Order("sku asc").Limit(limit)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"sku ILIKE ? OR asin ILIKE ? OR spu ILIKE ? OR name ILIKE ? OR style_name ILIKE ? OR color ILIKE ? OR category1 ILIKE ? OR category2 ILIKE ? OR category3 ILIKE ?",
			like, like, like, like, like, like, like, like, like,
		)
	}

	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SKUs"})
		return
	}
	hydrateProductPlatformSKUs(items)
	c.JSON(http.StatusOK, items)
}

func AdminListProductSKUs(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	query := productSKUQuery(keyword)

	var total int64
	if err := query.Model(&model.ProductSKU{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count product SKUs"})
		return
	}

	var items []model.ProductSKU
	if err := query.Order("sku asc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SKUs"})
		return
	}
	hydrateProductPlatformSKUs(items)

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func AdminCreateProductSKU(c *gin.Context) {
	item, ok := bindProductSKUInput(c)
	if !ok {
		return
	}
	if item.SPUID == 0 {
		spu, err := getProductSPUByName(item.SPU)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve product SPU"})
			return
		}
		if spu.ID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SPU is required"})
			return
		}
		item.SPUID = spu.ID
	}
	item.UpdatedBy = currentUserID(c)
	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product SKU: " + err.Error()})
		return
	}
	touchProductSPU(item.SPUID, item.UpdatedBy)
	c.JSON(http.StatusCreated, item)
}

func AdminUpdateProductSKU(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var existing model.ProductSKU
	if err := database.DB.First(&existing, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product SKU not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SKU"})
		return
	}
	previousSPUID := existing.SPUID

	item, ok := bindProductSKUInput(c)
	if !ok {
		return
	}
	if item.SPUID == 0 {
		spu, err := getProductSPUByName(item.SPU)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve product SPU"})
			return
		}
		if spu.ID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SPU is required"})
			return
		}
		item.SPUID = spu.ID
	}
	existing.SKU = item.SKU
	existing.ASIN = item.ASIN
	existing.ImageURL = item.ImageURL
	existing.ImageURL = item.ImageURL
	existing.Name = item.Name
	existing.SPU = item.SPU
	existing.SPUID = item.SPUID
	existing.StyleName = item.StyleName
	existing.Color = item.Color
	existing.Size = item.Size
	existing.Feature = item.Feature
	existing.PackageQty = item.PackageQty
	existing.Material = item.Material
	existing.Category1 = item.Category1
	existing.Category2 = item.Category2
	existing.Category3 = item.Category3
	existing.ListedAt = item.ListedAt
	existing.PackageLength = item.PackageLength
	existing.PackageWidth = item.PackageWidth
	existing.PackageHeight = item.PackageHeight
	existing.PackageWeight = item.PackageWeight
	existing.PurchasePrice = item.PurchasePrice
	existing.StpModelURL = item.StpModelURL
	existing.AuthDocURL = item.AuthDocURL
	existing.UpdatedBy = currentUserID(c)

	if err := database.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product SKU: " + err.Error()})
		return
	}
	touchProductSPU(previousSPUID, existing.UpdatedBy)
	if existing.SPUID != previousSPUID {
		touchProductSPU(existing.SPUID, existing.UpdatedBy)
	}
	c.JSON(http.StatusOK, existing)
}

type productPlatformSKUInput struct {
	PlatformSKUs map[string][]string `json:"platform_skus"`
}

// UpdateProductSKUPlatformSKUs keeps marketplace aliases in the same product
// editing workflow while retaining the existing mapping table as the source
// used by order conversion.
func UpdateProductSKUPlatformSKUs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var productSKU model.ProductSKU
	if err := database.DB.First(&productSKU, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product SKU not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product SKU"})
		return
	}
	var input productPlatformSKUInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	platformSKUs := make(map[string][]string)
	for rawPlatform, values := range input.PlatformSKUs {
		platform := normalizeOrderPlatform(rawPlatform)
		if platform == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
			return
		}
		if !canEditOrderPlatform(user, platform) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You cannot edit mappings for this platform"})
			return
		}
		platformSKUs[platform] = []string{}
		seen := make(map[string]struct{})
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			platformSKUs[platform] = append(platformSKUs[platform], value)
		}
	}
	now := time.Now()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for platform, values := range platformSKUs {
			if err := tx.Model(&model.OrderSKUMapping{}).
				Where("platform = ? AND amazon_sku = ? AND status = 'active'", platform, productSKU.SKU).
				Updates(map[string]interface{}{"status": "inactive", "is_current": false, "updated_by": user.ID, "updated_at": now}).Error; err != nil {
				return err
			}
			for _, platformSKU := range values {
				if err := tx.Model(&model.OrderSKUMapping{}).
					Where("platform = ? AND platform_sku = ? AND status = 'active'", platform, platformSKU).
					Update("is_current", false).Error; err != nil {
					return err
				}
				item := model.OrderSKUMapping{Platform: platform, PlatformSKU: platformSKU, AmazonSKU: productSKU.SKU, Status: "active", IsCurrent: true, CreatedBy: user.ID, UpdatedBy: user.ID, CreatedAt: now, UpdatedAt: now}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "platform"}, {Name: "platform_sku"}, {Name: "amazon_sku"}},
					DoUpdates: clause.Assignments(map[string]interface{}{"status": "active", "is_current": true, "updated_by": user.ID, "updated_at": now}),
				}).Create(&item).Error; err != nil {
					return err
				}
				if err := reconcileCurrentOrderSKUMappings(tx, platform); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save platform SKUs"})
		return
	}
	item := productSKU
	hydrateProductPlatformSKUs([]model.ProductSKU{item})
	c.JSON(http.StatusOK, item)
}

func AdminDeleteProductSKU(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var existing model.ProductSKU
	if err := database.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product SKU not found"})
		return
	}
	if err := database.DB.Delete(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product SKU"})
		return
	}
	touchProductSPU(existing.SPUID, currentUserID(c))
	c.JSON(http.StatusOK, gin.H{"message": "Product SKU deleted successfully"})
}

type productSKUInput struct {
	ImageURL      string `json:"image_url"`
	SPUID         uint   `json:"spu_id"`
	SKU           string `json:"sku"`
	ASIN          string `json:"asin"`
	Name          string `json:"name"`
	SPU           string `json:"spu"`
	StyleName     string `json:"style_name"`
	Color         string `json:"color"`
	Size          string `json:"size"`
	Feature       string `json:"feature"`
	PackageQty    string `json:"package_qty"`
	Material      string `json:"material"`
	Category1     string `json:"category1"`
	Category2     string `json:"category2"`
	Category3     string `json:"category3"`
	ListedAt      string `json:"listed_at"`
	PackageLength string `json:"package_length"`
	PackageWidth  string `json:"package_width"`
	PackageHeight string `json:"package_height"`
	PackageWeight string `json:"package_weight"`
	PurchasePrice string `json:"purchase_price"`
	StpModelURL   string `json:"stp_model_url"`
	AuthDocURL    string `json:"auth_doc_url"`
}

func bindProductSKUInput(c *gin.Context) (model.ProductSKU, bool) {
	var input productSKUInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return model.ProductSKU{}, false
	}
	item := model.ProductSKU{
		SPUID:         input.SPUID,
		ImageURL:      strings.TrimSpace(input.ImageURL),
		SKU:           strings.TrimSpace(input.SKU),
		ASIN:          strings.TrimSpace(input.ASIN),
		Name:          strings.TrimSpace(input.Name),
		SPU:           strings.TrimSpace(input.SPU),
		StyleName:     strings.TrimSpace(input.StyleName),
		Color:         strings.TrimSpace(input.Color),
		Size:          strings.TrimSpace(input.Size),
		Feature:       strings.TrimSpace(input.Feature),
		PackageQty:    strings.TrimSpace(input.PackageQty),
		Material:      strings.TrimSpace(input.Material),
		Category1:     strings.TrimSpace(input.Category1),
		Category2:     strings.TrimSpace(input.Category2),
		Category3:     strings.TrimSpace(input.Category3),
		ListedAt:      strings.TrimSpace(input.ListedAt),
		PackageLength: strings.TrimSpace(input.PackageLength),
		PackageWidth:  strings.TrimSpace(input.PackageWidth),
		PackageHeight: strings.TrimSpace(input.PackageHeight),
		PackageWeight: strings.TrimSpace(input.PackageWeight),
		PurchasePrice: strings.TrimSpace(input.PurchasePrice),
		StpModelURL:   strings.TrimSpace(input.StpModelURL),
		AuthDocURL:    strings.TrimSpace(input.AuthDocURL),
	}
	if item.SKU == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SKU is required"})
		return model.ProductSKU{}, false
	}
	if item.ASIN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ASIN is required"})
		return model.ProductSKU{}, false
	}
	if item.SPU == "" && item.SPUID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SPU is required"})
		return model.ProductSKU{}, false
	}
	return item, true
}

func productSKUQuery(keyword string) *gorm.DB {
	query := database.DB.Model(&model.ProductSKU{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"sku ILIKE ? OR asin ILIKE ? OR spu ILIKE ? OR name ILIKE ? OR style_name ILIKE ? OR color ILIKE ? OR size ILIKE ? OR category1 ILIKE ? OR category2 ILIKE ? OR category3 ILIKE ?",
			like, like, like, like, like, like, like, like, like, like,
		)
	}
	return query
}

func hydrateProductPlatformSKUs(items []model.ProductSKU) {
	if len(items) == 0 {
		return
	}
	amazonSKUs := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.SKU) != "" {
			amazonSKUs = append(amazonSKUs, item.SKU)
		}
	}
	var mappings []model.OrderSKUMapping
	if len(amazonSKUs) > 0 {
		_ = database.DB.Select("platform", "platform_sku", "amazon_sku").
			Where("amazon_sku IN ? AND status = 'active' AND is_current = true", amazonSKUs).
			Order("platform asc, platform_sku asc").Find(&mappings).Error
	}
	grouped := make(map[string]map[string][]string)
	for _, mapping := range mappings {
		if grouped[mapping.AmazonSKU] == nil {
			grouped[mapping.AmazonSKU] = make(map[string][]string)
		}
		grouped[mapping.AmazonSKU][mapping.Platform] = append(grouped[mapping.AmazonSKU][mapping.Platform], mapping.PlatformSKU)
	}
	for index := range items {
		items[index].PlatformSKUs = grouped[items[index].SKU]
		if items[index].PlatformSKUs == nil {
			items[index].PlatformSKUs = map[string][]string{}
		}
		for platform := range items[index].PlatformSKUs {
			sort.Strings(items[index].PlatformSKUs[platform])
		}
	}
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getProductSPUByName(spuName string) (model.ProductSPU, error) {
	spuName = strings.TrimSpace(spuName)
	var spu model.ProductSPU
	if spuName == "" {
		return spu, nil
	}
	if err := database.DB.Where("spu = ?", spuName).First(&spu).Error; err == nil {
		return spu, nil
	}
	return model.ProductSPU{}, gorm.ErrRecordNotFound
}
