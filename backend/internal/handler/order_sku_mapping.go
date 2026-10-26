package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxResolvedOrderSKUs = 5000
const maxImportedOrderSKUs = 2000

var errPriceMappingExists = errors.New("该商品或 TK SKU 已有映射，请在订单转 MCF 中维护")

type orderSKUMappingResolveInput struct {
	Platform string   `json:"platform"`
	SKUs     []string `json:"skus"`
}

type orderSKUMappingInput struct {
	Platform     string `json:"platform"`
	PlatformSKU  string `json:"platform_sku"`
	AmazonSKU    string `json:"amazon_sku"`
	NonAmazonSKU string `json:"non_amazon_sku"`
	Remark       string `json:"remark"`
	IsCurrent    *bool  `json:"is_current"`
	OnlyUnmapped bool   `json:"only_unmapped"`
}

type orderSKUMappingImportInput struct {
	Platform string                 `json:"platform"`
	Mode     string                 `json:"mode"`
	Items    []orderSKUMappingInput `json:"items"`
}

func ImportOrderSKUMappings(c *gin.Context) {
	var input orderSKUMappingImportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	platform := normalizeOrderPlatform(input.Platform)
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}
	mode := normalizeOrderSKUMappingImportMode(input.Mode)
	if mode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Import mode must be create or update"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	if !canEditOrderPlatform(user, platform) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot edit mappings for this platform"})
		return
	}

	items, err := normalizeImportedOrderSKUs(input.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(items) == 0 || len(items) > maxImportedOrderSKUs {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Import must contain 1 to 2000 mappings"})
		return
	}

	amazonSKUs := make([]string, 0, len(items))
	for _, item := range items {
		amazonSKUs = append(amazonSKUs, item.AmazonSKU)
	}
	var existing []string
	if err := database.DB.Model(&model.ProductSKU{}).Where("sku IN ?", amazonSKUs).Pluck("sku", &existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate Amazon SKUs"})
		return
	}
	validItems, missing := partitionImportedOrderSKUsByAmazonLibrary(items, existing)
	if len(validItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some Amazon SKUs do not exist", "missing_skus": missing})
		return
	}
	skipped := len(items) - len(validItems)
	items = validItems

	platformSKUs := make([]string, 0, len(items))
	for _, item := range items {
		platformSKUs = append(platformSKUs, item.PlatformSKU)
	}
	var existingMappingKeys []string
	if err := database.DB.Model(&model.OrderSKUMapping{}).
		Where("platform = ? AND platform_sku IN ?", platform, platformSKUs).
		Select("platform_sku || E'\\x00' || amazon_sku").Scan(&existingMappingKeys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate platform SKUs"})
		return
	}
	existingMappingSet := make(map[string]struct{}, len(existingMappingKeys))
	for _, key := range existingMappingKeys {
		existingMappingSet[key] = struct{}{}
	}
	items, skippedPlatformSKUs := partitionImportedOrderSKUsByMode(items, existingMappingSet, mode)

	now := time.Now()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for _, inputItem := range items {
			if mode == "update" {
				var existingItem model.OrderSKUMapping
				if findErr := tx.Where("platform = ? AND platform_sku = ? AND amazon_sku = ?", platform, inputItem.PlatformSKU, inputItem.AmazonSKU).First(&existingItem).Error; findErr != nil {
					return findErr
				}
				keepCurrent := existingItem.IsCurrent && existingItem.AmazonSKU == inputItem.AmazonSKU
				if updateErr := tx.Model(&model.OrderSKUMapping{}).
					Where("platform = ? AND platform_sku = ? AND amazon_sku = ?", platform, inputItem.PlatformSKU, inputItem.AmazonSKU).
					Updates(map[string]interface{}{
						"amazon_sku":     inputItem.AmazonSKU,
						"non_amazon_sku": inputItem.NonAmazonSKU,
						"remark":         inputItem.Remark,
						"is_current":     keepCurrent,
						"status":         "active",
						"updated_by":     user.ID,
						"updated_at":     now,
					}).Error; updateErr != nil {
					return updateErr
				}
				continue
			}
			item := model.OrderSKUMapping{
				Platform:     platform,
				PlatformSKU:  inputItem.PlatformSKU,
				AmazonSKU:    inputItem.AmazonSKU,
				NonAmazonSKU: inputItem.NonAmazonSKU,
				Remark:       inputItem.Remark,
				Status:       "active",
				CreatedBy:    user.ID,
				UpdatedBy:    user.ID,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if createErr := tx.Create(&item).Error; createErr != nil {
				return createErr
			}
		}
		if reconcileErr := reconcileCurrentOrderSKUMappings(tx, platform); reconcileErr != nil {
			return reconcileErr
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import SKU mappings"})
		return
	}
	existingSkipped := []string{}
	missingSkipped := []string{}
	if mode == "create" {
		existingSkipped = skippedPlatformSKUs
	} else {
		missingSkipped = skippedPlatformSKUs
	}
	c.JSON(http.StatusOK, gin.H{
		"imported":               len(items),
		"mode":                   mode,
		"skipped":                skipped + len(skippedPlatformSKUs),
		"missing_skus":           missing,
		"existing_platform_skus": existingSkipped,
		"missing_platform_skus":  missingSkipped,
	})
}

func partitionImportedOrderSKUsByMode(items []orderSKUMappingInput, existing map[string]struct{}, mode string) ([]orderSKUMappingInput, []string) {
	eligible := make([]orderSKUMappingInput, 0, len(items))
	skipped := make([]string, 0)
	for _, item := range items {
		key := item.PlatformSKU + "\x00" + item.AmazonSKU
		if item.AmazonSKU == "" {
			key = item.PlatformSKU
		}
		_, exists := existing[key]
		if (mode == "create" && !exists) || (mode == "update" && exists) {
			eligible = append(eligible, item)
			continue
		}
		skipped = append(skipped, item.PlatformSKU)
	}
	return eligible, skipped
}

func partitionImportedOrderSKUsByAmazonLibrary(items []orderSKUMappingInput, existingSKUs []string) ([]orderSKUMappingInput, []string) {
	existingSet := make(map[string]struct{}, len(existingSKUs))
	for _, sku := range existingSKUs {
		existingSet[sku] = struct{}{}
	}
	valid := make([]orderSKUMappingInput, 0, len(items))
	missing := make([]string, 0)
	missingSet := make(map[string]struct{})
	for _, item := range items {
		if _, exists := existingSet[item.AmazonSKU]; exists {
			valid = append(valid, item)
			continue
		}
		if _, seen := missingSet[item.AmazonSKU]; !seen {
			missingSet[item.AmazonSKU] = struct{}{}
			missing = append(missing, item.AmazonSKU)
		}
	}
	return valid, missing
}

func normalizeOrderSKUMappingImportMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "create":
		return "create"
	case "update":
		return "update"
	default:
		return ""
	}
}

func normalizeImportedOrderSKUs(input []orderSKUMappingInput) ([]orderSKUMappingInput, error) {
	seen := make(map[string]orderSKUMappingInput, len(input))
	result := make([]orderSKUMappingInput, 0, len(input))
	for index, item := range input {
		platformSKU := strings.TrimSpace(item.PlatformSKU)
		amazonSKU := strings.TrimSpace(item.AmazonSKU)
		nonAmazonSKU := strings.TrimSpace(item.NonAmazonSKU)
		remark := strings.TrimSpace(item.Remark)
		if len([]rune(remark)) > 1000 {
			return nil, newOrderMappingError("Row " + strconv.Itoa(index+1) + " has a remark longer than 1000 characters")
		}
		if platformSKU == "" || amazonSKU == "" || nonAmazonSKU == "" {
			return nil, newOrderMappingError("Row " + strconv.Itoa(index+1) + " has an empty SKU")
		}
		key := platformSKU + "\x00" + amazonSKU
		if existing, exists := seen[key]; exists {
			if existing.AmazonSKU != amazonSKU || existing.NonAmazonSKU != nonAmazonSKU || existing.Remark != remark {
				return nil, newOrderMappingError("Conflicting mappings for platform SKU: " + platformSKU)
			}
			continue
		}
		normalized := orderSKUMappingInput{PlatformSKU: platformSKU, AmazonSKU: amazonSKU, NonAmazonSKU: nonAmazonSKU, Remark: remark}
		seen[key] = normalized
		result = append(result, normalized)
	}
	return result, nil
}

func ListOrderSKUMappings(c *gin.Context) {
	platform := normalizeOrderPlatform(c.Query("platform"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	query := database.DB.Model(&model.OrderSKUMapping{}).
		Where("platform = ? AND status = ? AND is_current = true", platform, "active")
	if c.Query("duplicates_only") == "true" {
		query = query.Where(`platform_sku IN (
			SELECT platform_sku FROM order_sku_mappings
			WHERE platform = ? AND status = 'active'
			GROUP BY platform_sku HAVING COUNT(*) > 1
		)`, platform)
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where(`LOWER(platform_sku) LIKE ? OR LOWER(amazon_sku) LIKE ? OR LOWER(non_amazon_sku) LIKE ? OR LOWER(remark) LIKE ? OR EXISTS (
			SELECT 1 FROM order_sku_mappings related
			WHERE related.platform = order_sku_mappings.platform
				AND related.platform_sku = order_sku_mappings.platform_sku
				AND related.status = 'active'
				AND (LOWER(related.amazon_sku) LIKE ? OR LOWER(related.non_amazon_sku) LIKE ? OR LOWER(related.remark) LIKE ?)
		)`, like, like, like, like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count SKU mappings"})
		return
	}
	items := make([]model.OrderSKUMapping, 0)
	if err := query.Order("updated_at desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SKU mappings"})
		return
	}
	hydrateOrderMappingAuditNames(items)
	hydrateOrderMappingGroups(items)
	var duplicateGroups int64
	database.DB.Raw(`SELECT COUNT(*) FROM (
			SELECT platform_sku FROM order_sku_mappings
			WHERE platform = ? AND status = 'active'
			GROUP BY platform_sku HAVING COUNT(*) > 1
	) AS duplicate_groups`, platform).Scan(&duplicateGroups)
	c.JSON(http.StatusOK, gin.H{
		"items":            items,
		"total":            total,
		"page":             page,
		"page_size":        pageSize,
		"can_edit":         canEditOrderPlatform(user, platform),
		"duplicate_groups": duplicateGroups,
	})
}

func GetOrderSKUMappingGroup(c *gin.Context) {
	platform := normalizeOrderPlatform(c.Query("platform"))
	platformSKU := strings.TrimSpace(c.Query("platform_sku"))
	if platform == "" || platformSKU == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Platform and platform SKU are required"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	items := make([]model.OrderSKUMapping, 0)
	if err := database.DB.Where("platform = ? AND platform_sku = ? AND status = 'active'", platform, platformSKU).
		Order("is_current desc, amazon_sku asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SKU mapping group"})
		return
	}
	hydrateOrderMappingAuditNames(items)
	hydrateOrderMappingGroups(items)
	c.JSON(http.StatusOK, gin.H{"items": items, "can_edit": canEditOrderPlatform(user, platform)})
}

func ResolveOrderSKUMappings(c *gin.Context) {
	var input orderSKUMappingResolveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	platform := normalizeOrderPlatform(input.Platform)
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}

	skus := compactOrderSKUs(input.SKUs)
	if len(skus) > maxResolvedOrderSKUs {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many SKUs"})
		return
	}

	items := make([]model.OrderSKUMapping, 0)
	if len(skus) > 0 {
		if err := database.DB.Where("platform = ? AND platform_sku IN ? AND status = ? AND is_current = true", platform, skus, "active").
			Order("platform_sku asc").Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SKU mappings"})
			return
		}
	}
	hydrateOrderMappingAuditNames(items)
	hydrateOrderMappingGroups(items)

	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"can_edit": canEditOrderPlatform(user, platform),
	})
}

func UpsertOrderSKUMapping(c *gin.Context) {
	var input orderSKUMappingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	platform := normalizeOrderPlatform(input.Platform)
	platformSKU := strings.TrimSpace(input.PlatformSKU)
	amazonSKU := strings.TrimSpace(input.AmazonSKU)
	nonAmazonSKU := strings.TrimSpace(input.NonAmazonSKU)
	remark := strings.TrimSpace(input.Remark)
	if len([]rune(remark)) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Remark is too long"})
		return
	}
	if platform == "" || platformSKU == "" || amazonSKU == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Platform, platform SKU and Amazon SKU are required"})
		return
	}

	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	if !canEditOrderPlatform(user, platform) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot edit mappings for this platform"})
		return
	}

	var productSKU model.ProductSKU
	if err := database.DB.Where("sku = ?", amazonSKU).First(&productSKU).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amazon SKU does not exist in the product SKU library"})
		return
	}

	now := time.Now()
	item := model.OrderSKUMapping{
		Platform:     platform,
		PlatformSKU:  platformSKU,
		AmazonSKU:    productSKU.SKU,
		NonAmazonSKU: nonAmazonSKU,
		Remark:       remark,
		Status:       "active",
		CreatedBy:    user.ID,
		UpdatedBy:    user.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if input.OnlyUnmapped {
			// Serialize this check with mapping writes, including order conversion
			// edits, so a stale price page cannot replace or extend a mapping.
			if err := tx.Exec("LOCK TABLE order_sku_mappings IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
				return err
			}
			var count int64
			if err := tx.Model(&model.OrderSKUMapping{}).
				Where("platform = ? AND (platform_sku = ? OR (amazon_sku = ? AND status = 'active' AND is_current = true))", platform, platformSKU, amazonSKU).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errPriceMappingExists
			}
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "platform"}, {Name: "platform_sku"}, {Name: "amazon_sku"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"amazon_sku":     item.AmazonSKU,
				"non_amazon_sku": item.NonAmazonSKU,
				"remark":         item.Remark,
				"status":         "active",
				"updated_by":     user.ID,
				"updated_at":     now,
			}),
		}).Create(&item).Error; err != nil {
			return err
		}
		if err := tx.Where("platform = ? AND platform_sku = ? AND amazon_sku = ?", platform, platformSKU, item.AmazonSKU).First(&item).Error; err != nil {
			return err
		}
		if input.IsCurrent != nil && *input.IsCurrent {
			if err := tx.Model(&model.OrderSKUMapping{}).
				Where("platform = ? AND platform_sku = ? AND status = 'active'", platform, item.PlatformSKU).
				Update("is_current", false).Error; err != nil {
				return err
			}
			if err := tx.Model(&item).Update("is_current", true).Error; err != nil {
				return err
			}
		}
		return reconcileCurrentOrderSKUMappings(tx, platform)
	})
	if errors.Is(err, errPriceMappingExists) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save SKU mapping"})
		return
	}
	if err := database.DB.Where("platform = ? AND platform_sku = ? AND amazon_sku = ?", platform, platformSKU, item.AmazonSKU).First(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload SKU mapping"})
		return
	}
	items := []model.OrderSKUMapping{item}
	hydrateOrderMappingAuditNames(items)
	hydrateOrderMappingGroups(items)
	item = items[0]
	c.JSON(http.StatusOK, item)
}

func DeactivateOrderSKUMapping(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var item model.OrderSKUMapping
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SKU mapping not found"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	if !canEditOrderPlatform(user, item.Platform) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot edit mappings for this platform"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&item).Updates(map[string]interface{}{
			"status":     "inactive",
			"is_current": false,
			"updated_by": user.ID,
		}).Error; err != nil {
			return err
		}
		return reconcileCurrentOrderSKUMappings(tx, item.Platform)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate SKU mapping"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SKU mapping deactivated"})
}

func reconcileCurrentOrderSKUMappings(tx *gorm.DB, platform string) error {
	var items []model.OrderSKUMapping
	if err := tx.Where("platform = ? AND status = 'active'", platform).Order("amazon_sku asc, id asc").Find(&items).Error; err != nil {
		return err
	}
	groups := make(map[string][]model.OrderSKUMapping)
	for _, item := range items {
		groups[item.PlatformSKU] = append(groups[item.PlatformSKU], item)
	}
	for _, group := range groups {
		currentID := uint(0)
		for _, item := range group {
			if item.IsCurrent && currentID == 0 {
				currentID = item.ID
			}
		}
		if currentID == 0 {
			currentID = group[0].ID
		}
		if err := tx.Model(&model.OrderSKUMapping{}).
			Where("platform = ? AND platform_sku = ? AND status = 'active' AND is_current = true", platform, group[0].PlatformSKU).
			UpdateColumn("is_current", false).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.OrderSKUMapping{}).
			Where("id = ? AND is_current = false", currentID).
			UpdateColumn("is_current", true).Error; err != nil {
			return err
		}
	}
	return nil
}

func hydrateOrderMappingGroups(items []model.OrderSKUMapping) {
	if len(items) == 0 {
		return
	}
	platforms := make([]string, 0, len(items))
	platformSKUs := make([]string, 0, len(items))
	for _, item := range items {
		platforms = append(platforms, item.Platform)
		platformSKUs = append(platformSKUs, item.PlatformSKU)
	}
	var related []model.OrderSKUMapping
	if err := database.DB.Select("platform", "platform_sku", "amazon_sku", "is_current").
		Where("platform IN ? AND platform_sku IN ? AND status = 'active'", compactOrderSKUs(platforms), compactOrderSKUs(platformSKUs)).
		Order("platform_sku asc, amazon_sku asc").Find(&related).Error; err != nil {
		return
	}
	type groupInfo struct {
		SKUs    []string
		Current string
	}
	groups := make(map[string]*groupInfo)
	for _, item := range related {
		key := item.Platform + "\x00" + item.PlatformSKU
		if groups[key] == nil {
			groups[key] = &groupInfo{}
		}
		groups[key].SKUs = append(groups[key].SKUs, item.AmazonSKU)
		if item.IsCurrent {
			groups[key].Current = item.AmazonSKU
		}
	}
	for index := range items {
		group := groups[items[index].Platform+"\x00"+items[index].PlatformSKU]
		if group == nil {
			continue
		}
		items[index].RelatedAmazonSKUs = group.SKUs
		items[index].CurrentAmazonSKU = group.Current
		items[index].SelectionRequired = !items[index].IsCurrent
	}
}

func getOrderMappingUser(c *gin.Context) (model.User, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return model.User{}, false
	}
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return model.User{}, false
	}
	loadUserOrderPlatforms(&user)
	return user, true
}

func canEditOrderPlatform(user model.User, platform string) bool {
	if isOrderAdmin(user) {
		return true
	}
	return userResponsibleForOrderPlatform(user, platform)
}

func userResponsibleForOrderPlatform(user model.User, platform string) bool {
	platform = normalizeOrderPlatform(platform)
	for _, assigned := range user.OrderPlatforms {
		if normalizeOrderPlatform(assigned) == platform {
			return true
		}
	}
	return !user.OrderPlatformsConfigured && len(user.OrderPlatforms) == 0 && normalizeOrderPlatform(user.Platform) == platform
}

func userOrderPlatforms(user model.User) []string {
	if len(user.OrderPlatforms) > 0 {
		return normalizeUserOrderPlatforms(user.OrderPlatforms)
	}
	if !user.OrderPlatformsConfigured {
		if platform := normalizeOrderPlatform(user.Platform); platform != "" {
			return []string{platform}
		}
	}
	return nil
}

func isOrderAdmin(user model.User) bool {
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

func normalizeOrderPlatform(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "TEMU":
		return "TEMU"
	case "TIKTOK", "TIKTOK SHOP", "TK":
		return "TIKTOK"
	case "LOWES", "LOWE'S", "LOWE’S", "LOWE":
		return "LOWES"
	default:
		return ""
	}
}

func compactOrderSKUs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func hydrateOrderMappingAuditNames(items []model.OrderSKUMapping) {
	userIDs := make([]uint, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.UpdatedBy)
	}
	names := loadUserDisplayNames(userIDs)
	for i := range items {
		items[i].UpdatedByName = names[items[i].UpdatedBy]
	}
}
