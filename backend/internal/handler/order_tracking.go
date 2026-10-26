package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxImportedOrderTrackingItems = 20000
const maxMCFExcludedNotifications = 20000
const maxMCFEmailBytes = 10 << 20

const (
	orderStatusPendingShipment = "pending_shipment"
	orderStatusShipped         = "shipped"
	orderStatusCancelRequested = "cancel_requested"
	orderStatusCancelled       = "cancelled"
)

type orderTrackingImportInput struct {
	Platform string                   `json:"platform"`
	Items    []orderTrackingItemInput `json:"items"`
}

type orderTrackingGeneratedCheckInput struct {
	Platform string                   `json:"platform"`
	Items    []orderTrackingItemInput `json:"items"`
}

type mcfExcludedNotificationInput struct {
	Platform string                        `json:"platform"`
	Items    []mcfExcludedNotificationItem `json:"items"`
}

type mcfExcludedNotificationItem struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type orderTrackingItemInput struct {
	OrderID          string                      `json:"order_id"`
	PlatformSKU      string                      `json:"sku"`
	PackageKey       string                      `json:"package_key"`
	Quantity         int                         `json:"quantity"`
	Recipient        string                      `json:"recipient"`
	Phone            string                      `json:"phone"`
	Address          string                      `json:"address"`
	OriginalOrderIDs []string                    `json:"original_order_ids"`
	OriginalItems    []model.OrderTrackingSource `json:"original_items"`
	IsMergedOrder    bool                        `json:"is_merged_order"`
}

type orderTrackingUpdateInput struct {
	TrackingNumber string `json:"tracking_number"`
	AmazonOrderID  string `json:"amazon_order_id"`
	Remark         string `json:"remark"`
}

type orderTrackingStatusInput struct {
	Action string `json:"action"`
}

type orderTrackingExportInput struct {
	IDs []uint `json:"ids"`
}

type orderTrackingExportItem struct {
	Status         string                      `json:"status"`
	Details        []model.OrderTrackingDetail `json:"details"`
	Recipient      string                      `json:"recipient"`
	Phone          string                      `json:"phone"`
	Address        string                      `json:"address"`
	Remark         string                      `json:"remark"`
	ImportedBy     uint                        `json:"imported_by"`
	ImportedByName string                      `json:"imported_by_name"`
	CreatedAt      time.Time                   `json:"created_at"`
	ID             uint                        `json:"id"`
	Platform       string                      `json:"platform"`
	OrderID        string                      `json:"order_id"`
	PackageKey     string                      `json:"package_key"`
	TrackingNumber string                      `json:"tracking_number"`
	AmazonOrderID  string                      `json:"amazon_order_id"`
	Carrier        string                      `json:"carrier"`
	Exported       bool                        `json:"exported"`
}

func ImportOrderTrackingItems(c *gin.Context) {
	var input orderTrackingImportInput
	var history *model.OrderConversion
	var bindErr error
	if c.ContentType() == "multipart/form-data" {
		history, bindErr = readConversionUpload(c, &input)
	} else {
		bindErr = c.ShouldBindJSON(&input)
	}
	if bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}
	platform := normalizeOrderPlatform(input.Platform)
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}
	if len(input.Items) > maxImportedOrderTrackingItems {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many order items"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	if !canEditOrderPlatform(user, platform) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot import orders for this platform"})
		return
	}

	items, err := normalizeOrderTrackingItems(platform, user.ID, input.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"imported": 0})
		return
	}
	platformSKUs := make([]string, 0, len(items))
	for _, item := range items {
		platformSKUs = append(platformSKUs, item.PlatformSKU)
	}
	platformSKUs = compactOrderSKUs(platformSKUs)
	var mappedSKUs []string
	if err := database.DB.Model(&model.OrderSKUMapping{}).
		Where("platform = ? AND platform_sku IN ? AND status = ?", platform, platformSKUs, "active").
		Pluck("platform_sku", &mappedSKUs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate SKU mappings"})
		return
	}
	if missing := findUnmappedOrderTrackingSKUs(platformSKUs, mappedSKUs); len(missing) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order contains unmapped SKU: " + strings.Join(missing, ", ")})
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "platform"}, {Name: "order_id"}, {Name: "platform_sku"}, {Name: "package_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"quantity", "recipient", "phone", "address", "original_order_ids", "original_items", "is_merged_order", "updated_at"}),
		}).CreateInBatches(items, 500).Error; err != nil {
			return err
		}
		if history != nil {
			history.Platform = platform
			history.ImportedBy = user.ID
			input.Platform = platform
			history.Fingerprint = conversionFingerprint(history, input)
			return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "fingerprint"}}, DoNothing: true}).Create(history).Error
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "订单汇总或转换历史保存失败，请重试"})
		return
	}
	notifiedOrders := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := item.Platform + "\x00" + item.OrderID
		if _, exists := notifiedOrders[key]; exists {
			continue
		}
		notifiedOrders[key] = struct{}{}
		service.QueueOrderPendingShipmentNotification(item.Platform, item.OrderID)
	}
	c.JSON(http.StatusOK, gin.H{"imported": len(items)})
}

func CheckGeneratedOrderTrackingItems(c *gin.Context) {
	var input orderTrackingGeneratedCheckInput
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Items) > maxImportedOrderTrackingItems {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	platform := normalizeOrderPlatform(input.Platform)
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	accessiblePlatform, allowed := accessibleOrderTrackingPlatform(user, platform)
	if !allowed || accessiblePlatform != platform {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot view orders for this platform"})
		return
	}

	orderIDs, platformSKUs := compactOrderTrackingCheckValues(input.Items)
	if len(orderIDs) == 0 || len(platformSKUs) == 0 {
		c.JSON(http.StatusOK, gin.H{"items": []model.OrderTrackingItem{}})
		return
	}
	items := make([]model.OrderTrackingItem, 0)
	if err := database.DB.Select("order_id", "platform_sku", "original_order_ids").
		Where("platform = ? AND platform_sku IN ?", platform, platformSKUs).
		Where(`order_id IN ? OR EXISTS (
			SELECT 1 FROM jsonb_array_elements_text(original_order_ids) original_order_id
			WHERE original_order_id IN ?
		)`, orderIDs, orderIDs).
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check generated orders"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func NotifyMCFExcludedOrders(c *gin.Context) {
	var input mcfExcludedNotificationInput
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Items) > maxMCFExcludedNotifications {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	platform := normalizeOrderPlatform(input.Platform)
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	accessiblePlatform, allowed := accessibleOrderTrackingPlatform(user, platform)
	if !allowed || accessiblePlatform != platform {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot view orders for this platform"})
		return
	}
	notified := make(map[string]struct{}, len(input.Items))
	for _, item := range input.Items {
		orderID := strings.TrimSpace(item.OrderID)
		reason := strings.TrimSpace(item.Reason)
		if orderID == "" || reason == "" {
			continue
		}
		key := orderID + "\x00" + reason
		if _, exists := notified[key]; exists {
			continue
		}
		notified[key] = struct{}{}
		service.QueueOrderMCFExcludedNotification(platform, orderID, reason)
	}
	c.JSON(http.StatusOK, gin.H{"notified": len(notified)})
}

// SendMCFEmail sends a generated Lowe's MCF file as an email attachment.
func SendMCFEmail(c *gin.Context) {
	platform := normalizeOrderPlatform(c.PostForm("platform"))
	if platform != "LOWES" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MCF email is only supported for LOWES"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	accessiblePlatform, allowed := accessibleOrderTrackingPlatform(user, platform)
	if !allowed || accessiblePlatform != platform {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot view orders for this platform"})
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader.Size <= 0 || fileHeader.Size > maxMCFEmailBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid MCF file"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read MCF file"})
		return
	}
	defer file.Close()
	attachment, err := io.ReadAll(io.LimitReader(file, maxMCFEmailBytes+1))
	if err != nil || len(attachment) == 0 || len(attachment) > maxMCFEmailBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid MCF file"})
		return
	}
	fileName := strings.TrimSpace(fileHeader.Filename)
	if fileName == "" {
		fileName = "MCF_LOWES.txt"
	}
	if err := service.SendLowesMCFEmail(fileName, attachment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MCF 邮件发送失败：" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": true})
}

func compactOrderTrackingCheckValues(items []orderTrackingItemInput) ([]string, []string) {
	orderIDs := make([]string, 0, len(items))
	platformSKUs := make([]string, 0, len(items))
	for _, item := range items {
		if orderID := strings.TrimSpace(item.OrderID); orderID != "" {
			orderIDs = append(orderIDs, orderID)
		}
		if sku := strings.TrimSpace(item.PlatformSKU); sku != "" {
			platformSKUs = append(platformSKUs, sku)
		}
	}
	return compactOrderSKUs(orderIDs), compactOrderSKUs(platformSKUs)
}

func findUnmappedOrderTrackingSKUs(requested, mapped []string) []string {
	mappedSet := make(map[string]struct{}, len(mapped))
	for _, sku := range mapped {
		mappedSet[sku] = struct{}{}
	}
	missing := make([]string, 0)
	for _, sku := range requested {
		if _, exists := mappedSet[sku]; !exists {
			missing = append(missing, sku)
		}
	}
	return missing
}

func ListOrderTrackingItems(c *gin.Context) {
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	platform, allowed := accessibleOrderTrackingPlatform(user, c.Query("platform"))
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot view orders for this platform"})
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
	base := database.DB.Model(&model.OrderTrackingItem{})
	if platform != "" {
		base = base.Where("platform = ?", platform)
	} else if !canViewAllOrderTracking(user) {
		base = base.Where("platform IN ?", userOrderPlatforms(user))
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		base = base.Where(`LOWER(order_id) LIKE ? OR LOWER(platform_sku) LIKE ? OR LOWER(recipient) LIKE ? OR LOWER(address) LIKE ? OR LOWER(tracking_number) LIKE ? OR LOWER(remark) LIKE ? OR EXISTS (
			SELECT 1 FROM jsonb_array_elements_text(original_order_ids) original_order_id
			WHERE LOWER(original_order_id) LIKE ?
		) OR EXISTS (
			SELECT 1 FROM order_sku_mappings
			WHERE order_sku_mappings.platform = order_tracking_items.platform
				AND order_sku_mappings.platform_sku = order_tracking_items.platform_sku
				AND order_sku_mappings.status = 'active'
				AND LOWER(order_sku_mappings.amazon_sku) LIKE ?
		)`, like, like, like, like, like, like, like, like)
	}
	createdFrom, createdTo, err := parseOrderTrackingDateRange(c.Query("created_from"), c.Query("created_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if createdFrom != nil {
		base = base.Where("created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		base = base.Where("created_at < ?", *createdTo)
	}
	if status := normalizeOrderTrackingStatus(c.Query("status")); status != "" {
		base = base.Where("status = ?", status)
	}

	var total int64
	var pending int64
	var filled int64
	if total, err = countDistinctOrderTrackingOrders(base); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count orders"})
		return
	}
	if pending, err = countDistinctOrderTrackingOrders(base.Session(&gorm.Session{}).Where("tracking_number = '' AND status <> ?", orderStatusCancelled)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count pending orders"})
		return
	}
	if filled, err = countFullyTrackedOrderTrackingOrders(base.Session(&gorm.Session{}).Where("status <> ?", orderStatusCancelled)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count filled orders"})
		return
	}

	query := base.Session(&gorm.Session{})
	switch strings.ToLower(strings.TrimSpace(c.Query("tracking_status"))) {
	case "pending":
		query = query.Where("tracking_number = ''")
	case "filled":
		query = query.Where("tracking_number <> ''")
	}
	filteredTotal, err := countDistinctOrderTrackingOrders(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count filtered orders"})
		return
	}
	items := make([]model.OrderTrackingItem, 0)
	if err := query.Order("updated_at desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	hydrateOrderTrackingMetadata(items, user)
	items = collapseOrderTrackingItems(items)
	c.JSON(http.StatusOK, gin.H{
		"items":        items,
		"total":        filteredTotal,
		"page":         page,
		"page_size":    pageSize,
		"stats":        gin.H{"total": total, "pending": pending, "filled": filled},
		"platform":     platform,
		"is_admin":     isOrderAdmin(user),
		"can_view_all": canViewAllOrderTracking(user),
	})
}

func countDistinctOrderTrackingOrders(query *gorm.DB) (int64, error) {
	grouped := query.Session(&gorm.Session{}).
		Select("platform, order_id").
		Group("platform, order_id")
	var count int64
	err := database.DB.Table("(?) AS grouped_order_tracking", grouped).Count(&count).Error
	return count, err
}

func countFullyTrackedOrderTrackingOrders(query *gorm.DB) (int64, error) {
	grouped := query.Session(&gorm.Session{}).
		Select("platform, order_id").
		Group("platform, order_id").
		Having("bool_and(tracking_number <> '')")
	var count int64
	err := database.DB.Table("(?) AS fully_tracked_order_tracking", grouped).Count(&count).Error
	return count, err
}

func parseOrderTrackingDateRange(fromValue, toValue string) (*time.Time, *time.Time, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, nil, errors.New("failed to load date timezone")
	}
	parse := func(value string) (*time.Time, error) {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, nil
		}
		parsed, err := time.ParseInLocation("2006-01-02", value, location)
		if err != nil {
			return nil, errors.New("invalid added date")
		}
		return &parsed, nil
	}
	from, err := parse(fromValue)
	if err != nil {
		return nil, nil, err
	}
	to, err := parse(toValue)
	if err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, nil, errors.New("start date cannot be later than end date")
	}
	if to != nil {
		exclusiveTo := to.AddDate(0, 0, 1)
		to = &exclusiveTo
	}
	return from, to, nil
}

func collapseOrderTrackingItems(items []model.OrderTrackingItem) []model.OrderTrackingItem {
	result := make([]model.OrderTrackingItem, 0, len(items))
	indexes := make(map[string]int, len(items))
	packageIndexes := make(map[string]map[string]int, len(items))
	for _, item := range items {
		key := item.Platform + "\x00" + item.OrderID
		details := buildOrderTrackingDetails(item)
		if index, exists := indexes[key]; exists {
			result[index].Details = append(result[index].Details, details...)
			result[index].Quantity += item.Quantity
			appendOrderTrackingPackage(&result[index], item, packageIndexes[key])
			continue
		}
		item.Details = details
		indexes[key] = len(result)
		packageIndexes[key] = make(map[string]int)
		appendOrderTrackingPackage(&item, item, packageIndexes[key])
		result = append(result, item)
	}
	for index := range result {
		finalizeCollapsedOrder(&result[index])
	}
	return result
}

func appendOrderTrackingPackage(order *model.OrderTrackingItem, item model.OrderTrackingItem, indexes map[string]int) {
	packageKey := item.PackageKey
	if packageKey == "" {
		packageKey = "legacy"
	}
	if index, exists := indexes[packageKey]; exists {
		packageItem := &order.Packages[index]
		if packageItem.TrackingNumber == "" {
			packageItem.TrackingNumber = item.TrackingNumber
		}
		if order.AmazonOrderID == "" {
			order.AmazonOrderID = item.AmazonOrderID
		}
		if order.Phone == "" {
			order.Phone = item.Phone
		}
		if packageItem.AmazonOrderID == "" {
			packageItem.AmazonOrderID = item.AmazonOrderID
		}
		if packageItem.TrackingNumber != "" {
			packageItem.Status = orderStatusShipped
		}
		packageItem.Quantity += item.Quantity
		packageItem.Details = appendPackageDetails(packageItem.Details, buildOrderTrackingDetails(item))
		packageItem.Exported = packageItem.Exported || item.ExportedAt != nil
		packageItem.CanEditTracking = packageItem.CanEditTracking || item.CanEditTracking
		packageItem.CanRequestCancel = packageItem.CanRequestCancel || item.CanRequestCancel
		packageItem.CanCancel = packageItem.CanCancel || item.CanCancel
		packageItem.CanRevokeCancel = packageItem.CanRevokeCancel || item.CanRevokeCancel
		return
	}
	indexes[packageKey] = len(order.Packages)
	order.Packages = append(order.Packages, model.OrderTrackingPackage{
		ID: item.ID, PackageKey: packageKey, TrackingNumber: item.TrackingNumber,
		Quantity: item.Quantity,
		Carrier:  item.Carrier, Status: item.Status, Remark: item.Remark,
		AmazonOrderID: item.AmazonOrderID,
		Exported:      item.ExportedAt != nil, CanEditTracking: item.CanEditTracking,
		CanRequestCancel: item.CanRequestCancel, CanCancel: item.CanCancel, CanRevokeCancel: item.CanRevokeCancel,
		Details: buildOrderTrackingDetails(item),
	})
}

func appendPackageDetails(existing, incoming []model.OrderTrackingDetail) []model.OrderTrackingDetail {
	for _, detail := range incoming {
		found := false
		for index := range existing {
			if existing[index].OriginalOrderID == detail.OriginalOrderID && existing[index].AmazonSKU == detail.AmazonSKU {
				existing[index].Quantity += detail.Quantity
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, detail)
		}
	}
	return existing
}

func finalizeCollapsedOrder(order *model.OrderTrackingItem) {
	trackingNumbers := make([]string, 0, len(order.Packages))
	seen := make(map[string]struct{}, len(order.Packages))
	allShipped := len(order.Packages) > 0
	allCancelled := len(order.Packages) > 0
	anyCancelRequested := false
	for _, packageItem := range order.Packages {
		if packageItem.TrackingNumber != "" {
			if _, exists := seen[packageItem.TrackingNumber]; !exists {
				seen[packageItem.TrackingNumber] = struct{}{}
				trackingNumbers = append(trackingNumbers, packageItem.TrackingNumber)
			}
		} else {
			allShipped = false
		}
		if packageItem.Status != orderStatusCancelled {
			allCancelled = false
		}
		if packageItem.Status == orderStatusCancelRequested {
			anyCancelRequested = true
		}
	}
	order.TrackingNumber = strings.Join(trackingNumbers, "、")
	switch {
	case allCancelled:
		order.Status = orderStatusCancelled
	case anyCancelRequested:
		order.Status = orderStatusCancelRequested
	case allShipped:
		order.Status = orderStatusShipped
	default:
		order.Status = orderStatusPendingShipment
	}
	if len(order.Packages) > 0 {
		order.CanEditTracking = false
		order.CanRequestCancel = false
		order.CanCancel = false
		order.CanRevokeCancel = false
		for _, packageItem := range order.Packages {
			order.CanEditTracking = order.CanEditTracking || packageItem.CanEditTracking
			order.CanRequestCancel = order.CanRequestCancel || packageItem.CanRequestCancel
			order.CanCancel = order.CanCancel || packageItem.CanCancel
			order.CanRevokeCancel = order.CanRevokeCancel || packageItem.CanRevokeCancel
		}
	}
}

func buildOrderTrackingDetails(item model.OrderTrackingItem) []model.OrderTrackingDetail {
	if len(item.OriginalItems) == 0 {
		return []model.OrderTrackingDetail{{AmazonSKU: item.AmazonSKU, Quantity: item.Quantity}}
	}
	result := make([]model.OrderTrackingDetail, 0, len(item.OriginalItems))
	for _, source := range item.OriginalItems {
		amazonSKU := source.AmazonSKU
		if amazonSKU == "" && source.PlatformSKU == item.PlatformSKU {
			amazonSKU = item.AmazonSKU
		}
		result = append(result, model.OrderTrackingDetail{
			OriginalOrderID: source.OriginalOrderID,
			AmazonSKU:       amazonSKU,
			Quantity:        source.Quantity,
		})
	}
	return result
}

func ListExportableOrderTrackingItems(c *gin.Context) {
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	platform, allowed := accessibleOrderTrackingPlatform(user, c.Query("platform"))
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot view orders for this platform"})
		return
	}

	query := database.DB.Model(&model.OrderTrackingItem{}).
		Where("tracking_number <> '' AND status <> ?", orderStatusCancelled)
	if platform != "" {
		query = query.Where("platform = ?", platform)
	} else if !canViewAllOrderTracking(user) {
		query = query.Where("platform IN ?", userOrderPlatforms(user))
	}
	createdFrom, createdTo, err := parseOrderTrackingDateRange(c.Query("created_from"), c.Query("created_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if createdFrom != nil {
		query = query.Where("created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		query = query.Where("created_at < ?", *createdTo)
	}
	var items []model.OrderTrackingItem
	if err := query.Order("updated_at asc, id asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch exportable orders"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": buildOrderTrackingExportItems(items)})
}

func ExportOrderTrackingItems(c *gin.Context) {
	var input orderTrackingExportInput
	if err := c.ShouldBindJSON(&input); err != nil || len(input.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please select orders to export"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}

	var selected []model.OrderTrackingItem
	if err := database.DB.Where("id IN ? AND tracking_number <> '' AND status <> ?", input.IDs, orderStatusCancelled).Find(&selected).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch export orders"})
		return
	}
	allowed := make([]model.OrderTrackingItem, 0, len(selected))
	for _, item := range selected {
		if platform, accessible := accessibleOrderTrackingPlatform(user, item.Platform); accessible && platform == item.Platform {
			allowed = append(allowed, item)
		}
	}
	exportItems := buildOrderTrackingExportItems(allowed)
	if len(exportItems) == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "No exportable orders were selected"})
		return
	}
	// The selection contains one representative ID per package. Load all its SKU rows.
	packageKeys := make([][]interface{}, 0, len(exportItems))
	for _, item := range exportItems {
		packageKeys = append(packageKeys, []interface{}{item.Platform, item.OrderID, item.PackageKey})
	}
	if err := database.DB.Where("(platform, order_id, package_key) IN ?", packageKeys).
		Where("tracking_number <> '' AND status <> ?", orderStatusCancelled).
		Order("id asc").Find(&allowed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch export order details"})
		return
	}
	hydrateOrderTrackingMetadata(allowed, user)

	now := time.Now()
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range exportItems {
			if item.Exported {
				continue
			}
			updateQuery := tx.Model(&model.OrderTrackingItem{}).
				Where("platform = ? AND order_id = ? AND tracking_number <> '' AND exported_at IS NULL", item.Platform, item.OrderID)
			if item.PackageKey == "" {
				updateQuery = updateQuery.Where("package_key = ''")
			} else {
				updateQuery = updateQuery.Where("package_key = ?", item.PackageKey)
			}
			result := updateQuery.Update("exported_at", now)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Some selected orders have already been exported; please refresh and try again"})
		return
	}
	exportRows := buildOrderTrackingExportRows(allowed)
	c.JSON(http.StatusOK, gin.H{"items": exportRows, "exported": len(exportRows)})
}

func buildOrderTrackingExportItems(items []model.OrderTrackingItem) []orderTrackingExportItem {
	result := make([]orderTrackingExportItem, 0, len(items))
	seen := make(map[string]int, len(items))
	for _, item := range items {
		key := item.Platform + "\x00" + item.OrderID + "\x00" + item.PackageKey
		if index, exists := seen[key]; exists {
			result[index].Details = append(result[index].Details, buildOrderTrackingDetails(item)...)
			continue
		}
		seen[key] = len(result)
		result = append(result, buildOrderTrackingExportItem(item, item.OrderID))
	}
	return result
}

func buildOrderTrackingExportRows(items []model.OrderTrackingItem) []orderTrackingExportItem {
	orders := buildOrderTrackingExportItems(items)
	originalIDs := make(map[string][]string, len(items))
	for _, item := range items {
		key := item.Platform + "\x00" + item.OrderID + "\x00" + item.PackageKey
		if len(originalIDs[key]) == 0 && item.IsMergedOrder {
			originalIDs[key] = compactOrderSKUs(item.OriginalOrderIDs)
		}
	}
	result := make([]orderTrackingExportItem, 0, len(orders))
	for _, order := range orders {
		ids := originalIDs[order.Platform+"\x00"+order.OrderID+"\x00"+order.PackageKey]
		if len(ids) == 0 {
			result = append(result, order)
			continue
		}
		for _, originalID := range ids {
			row := order
			row.OrderID = originalID
			row.Details = make([]model.OrderTrackingDetail, 0)
			for _, detail := range order.Details {
				if detail.OriginalOrderID == originalID || detail.OriginalOrderID == "" {
					row.Details = append(row.Details, detail)
				}
			}
			result = append(result, row)
		}
	}
	return result
}

func buildOrderTrackingExportItem(item model.OrderTrackingItem, orderID string) orderTrackingExportItem {
	return orderTrackingExportItem{
		ID: item.ID, Platform: item.Platform, OrderID: orderID, PackageKey: item.PackageKey,
		TrackingNumber: item.TrackingNumber, Carrier: inferOrderCarrier(item.TrackingNumber), Exported: item.ExportedAt != nil,
		AmazonOrderID: item.AmazonOrderID,
		Status:        item.Status, Details: buildOrderTrackingDetails(item),
		Recipient: item.Recipient, Phone: item.Phone, Address: item.Address, Remark: item.Remark,
		ImportedBy: item.ImportedBy, ImportedByName: item.ImportedByName, CreatedAt: item.CreatedAt,
	}
}

func UpdateOrderTrackingNumber(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var input orderTrackingUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	trackingNumber := strings.TrimSpace(input.TrackingNumber)
	amazonOrderID := strings.TrimSpace(input.AmazonOrderID)
	remark := strings.TrimSpace(input.Remark)
	if len(trackingNumber) > 160 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tracking number is too long"})
		return
	}
	if len(amazonOrderID) > 160 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amazon order number is too long"})
		return
	}
	if utf8.RuneCountInString(remark) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Remark is too long"})
		return
	}

	var item model.OrderTrackingItem
	if err := database.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	if !canEditOrderTracking(user, item.Platform) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot update orders for this platform"})
		return
	}
	if item.Status == orderStatusCancelled || (item.Status == orderStatusCancelRequested && !canCancelOrder(user)) {
		c.JSON(http.StatusConflict, gin.H{"error": "Orders in the current status cannot update tracking information"})
		return
	}
	if trackingNumber != "" {
		conflicts, err := findTrackingNumberConflicts(item.Platform, item.OrderID, trackingNumber)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate tracking number"})
			return
		}
		if len(conflicts) > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error":              "Tracking number is already used by another order",
				"duplicate_tracking": true,
				"conflicts":          conflicts,
			})
			return
		}
	}

	now := time.Now()
	updates := map[string]interface{}{
		"tracking_number": trackingNumber,
		"amazon_order_id": amazonOrderID,
		"remark":          remark,
		"status":          orderTrackingStatusForNumber(trackingNumber),
		"updated_at":      now,
	}
	if trackingNumber != item.TrackingNumber {
		if trackingNumber == "" {
			updates["tracking_updated_by"] = 0
			updates["tracking_updated_at"] = nil
		} else {
			updates["tracking_updated_by"] = user.ID
			updates["tracking_updated_at"] = now
		}
	}
	updateQuery := database.DB.Model(&model.OrderTrackingItem{})
	if item.PackageKey == "" {
		updateQuery = updateQuery.Where("platform = ? AND order_id = ? AND package_key = ''", item.Platform, item.OrderID)
	} else {
		updateQuery = updateQuery.Where("id = ?", item.ID)
	}
	result := updateQuery.Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tracking number"})
		return
	}
	if item.TrackingNumber == "" && trackingNumber != "" {
		if item.Platform == "LOWES" && item.Status == orderStatusCancelRequested {
			service.QueueOrderCancelFailedTrackingNotification(item.Platform, item.OrderID)
		} else {
			service.QueueOrderTrackingNotification(item.Platform, item.OrderID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"updated": result.RowsAffected, "tracking_number": trackingNumber, "amazon_order_id": amazonOrderID, "remark": remark})
}

func UpdateOrderTrackingStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var input orderTrackingStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	var item model.OrderTrackingItem
	if err := database.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	nextStatus, allowed := nextOrderTrackingStatus(user, item, input.Action)
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot perform this status action"})
		return
	}
	updateQuery := database.DB.Model(&model.OrderTrackingItem{})
	if item.PackageKey == "" {
		updateQuery = updateQuery.Where("platform = ? AND order_id = ? AND package_key = ''", item.Platform, item.OrderID)
	} else {
		updateQuery = updateQuery.Where("id = ?", item.ID)
	}
	result := updateQuery.Updates(map[string]interface{}{"status": nextStatus, "updated_at": time.Now()})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}
	if nextStatus == orderStatusCancelled {
		service.QueueOrderCancelledNotification(item.Platform, item.OrderID)
	} else if nextStatus == orderStatusCancelRequested {
		service.QueueOrderCancelRequestedNotification(item.Platform, item.OrderID)
	}
	c.JSON(http.StatusOK, gin.H{"updated": result.RowsAffected, "status": nextStatus})
}

func nextOrderTrackingStatus(user model.User, item model.OrderTrackingItem, action string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "request_cancel":
		if !canRequestOrderCancellation(user, item.Platform) || item.Status != orderStatusPendingShipment {
			return "", false
		}
		return orderStatusCancelRequested, true
	case "cancel":
		if !canCancelOrder(user) || item.Status != orderStatusCancelRequested {
			return "", false
		}
		return orderStatusCancelled, true
	case "revoke_cancel":
		if !canRevokeOrderCancellation(user, item.Platform) || item.Status != orderStatusCancelRequested {
			return "", false
		}
		return orderStatusPendingShipment, true
	default:
		return "", false
	}
}

func normalizeOrderTrackingStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case orderStatusPendingShipment, orderStatusShipped, orderStatusCancelRequested, orderStatusCancelled:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func orderTrackingStatusForNumber(trackingNumber string) string {
	if strings.TrimSpace(trackingNumber) == "" {
		return orderStatusPendingShipment
	}
	return orderStatusShipped
}

func findTrackingNumberConflicts(platform, orderID, trackingNumber string) ([]model.OrderTrackingItem, error) {
	key := normalizeTrackingNumberKey(trackingNumber)
	if key == "" {
		return []model.OrderTrackingItem{}, nil
	}
	items := make([]model.OrderTrackingItem, 0)
	err := database.DB.Select("platform", "order_id", "tracking_number").
		Where("UPPER(REPLACE(tracking_number, ' ', '')) = ?", key).
		Where("NOT (platform = ? AND order_id = ?)", platform, orderID).
		Order("id asc").Limit(10).Find(&items).Error
	return items, err
}

func normalizeTrackingNumberKey(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
}

func normalizeOrderTrackingItems(platform string, userID uint, input []orderTrackingItemInput) ([]model.OrderTrackingItem, error) {
	now := time.Now()
	items := make([]model.OrderTrackingItem, 0, len(input))
	indexes := make(map[string]int, len(input))
	for index, raw := range input {
		orderID := strings.TrimSpace(raw.OrderID)
		sku := strings.TrimSpace(raw.PlatformSKU)
		packageKey := strings.TrimSpace(raw.PackageKey)
		recipient := strings.TrimSpace(raw.Recipient)
		phone := strings.TrimSpace(raw.Phone)
		address := strings.TrimSpace(raw.Address)
		if orderID == "" || sku == "" || raw.Quantity <= 0 {
			return nil, newOrderMappingError("Row " + strconv.Itoa(index+1) + " has invalid order data")
		}
		if strings.ContainsRune(packageKey, '\x00') {
			return nil, newOrderMappingError("Row " + strconv.Itoa(index+1) + " has an invalid package key")
		}
		if len(orderID) > 160 || len(sku) > 160 || len(packageKey) > 200 || len(recipient) > 200 || len(phone) > 80 {
			return nil, newOrderMappingError("Row " + strconv.Itoa(index+1) + " is too long")
		}
		key := orderID + "\x00" + sku + "\x00" + packageKey
		if existingIndex, exists := indexes[key]; exists {
			items[existingIndex].Quantity += raw.Quantity
			items[existingIndex].OriginalItems = append(items[existingIndex].OriginalItems, raw.OriginalItems...)
			continue
		}
		indexes[key] = len(items)
		items = append(items, model.OrderTrackingItem{
			Platform: platform, OrderID: orderID, PlatformSKU: sku, PackageKey: packageKey, Quantity: raw.Quantity,
			Recipient: recipient, Phone: phone, Address: address, OriginalOrderIDs: compactOrderSKUs(raw.OriginalOrderIDs), OriginalItems: raw.OriginalItems, IsMergedOrder: raw.IsMergedOrder,
			Status: orderStatusPendingShipment, ImportedBy: userID, CreatedAt: now, UpdatedAt: now,
		})
	}
	return items, nil
}

func accessibleOrderTrackingPlatform(user model.User, requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	if canViewAllOrderTracking(user) {
		if requested == "" {
			return "", true
		}
		platform := normalizeOrderPlatform(requested)
		return platform, platform != ""
	}
	platforms := userOrderPlatforms(user)
	if len(platforms) == 0 {
		return "", false
	}
	if requested == "" {
		if len(platforms) == 1 {
			return platforms[0], true
		}
		return "", true
	}
	requestedPlatform := normalizeOrderPlatform(requested)
	for _, platform := range platforms {
		if platform == requestedPlatform {
			return requestedPlatform, true
		}
	}
	return "", false
}

func hydrateOrderTrackingMetadata(items []model.OrderTrackingItem, user model.User) {
	userIDs := make([]uint, 0, len(items)*2)
	platforms := make([]string, 0, len(items))
	platformSKUs := make([]string, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.ImportedBy)
		userIDs = append(userIDs, item.TrackingUpdatedBy)
		platforms = append(platforms, item.Platform)
		platformSKUs = append(platformSKUs, item.PlatformSKU)
	}
	names := loadUserDisplayNames(userIDs)
	amazonSKUs := loadOrderTrackingAmazonSKUs(platforms, platformSKUs)
	for index := range items {
		items[index].ImportedByName = names[items[index].ImportedBy]
		items[index].TrackingUpdatedByName = names[items[index].TrackingUpdatedBy]
		items[index].AmazonSKU = amazonSKUs[items[index].Platform+"\x00"+items[index].PlatformSKU]
		items[index].Carrier = inferOrderCarrier(items[index].TrackingNumber)
		items[index].CanEditTracking = canEditOrderTracking(user, items[index].Platform) && items[index].Status != orderStatusCancelled && (items[index].Status != orderStatusCancelRequested || canCancelOrder(user))
		items[index].CanRequestCancel = canRequestOrderCancellation(user, items[index].Platform) && items[index].Status == orderStatusPendingShipment
		items[index].CanCancel = canCancelOrder(user) && items[index].Status == orderStatusCancelRequested
		items[index].CanRevokeCancel = canRevokeOrderCancellation(user, items[index].Platform) && items[index].Status == orderStatusCancelRequested
	}
}

func canViewAllOrderTracking(user model.User) bool {
	return isOrderAdmin(user) || isOrderDataUser(user)
}

func canEditOrderTracking(user model.User, platform string) bool {
	return isOrderDataUser(user) || canEditOrderPlatform(user, platform)
}

func canRequestOrderCancellation(user model.User, platform string) bool {
	return !isOrderDataUser(user) && userResponsibleForOrderPlatform(user, platform)
}

func canCancelOrder(user model.User) bool {
	return isOrderAdmin(user) || isOrderDataUser(user)
}

func canRevokeOrderCancellation(user model.User, platform string) bool {
	if normalizeOrderPlatform(platform) == "LOWES" {
		return false
	}
	return canCancelOrder(user) || canRequestOrderCancellation(user, platform)
}

func isOrderDataUser(user model.User) bool {
	platform := strings.ToLower(strings.TrimSpace(user.Platform))
	department := strings.ToLower(strings.TrimSpace(user.Department))
	return platform == "data" || department == "data" || department == "数据" || department == "数据部"
}

func loadOrderTrackingAmazonSKUs(platforms, platformSKUs []string) map[string]string {
	result := make(map[string]string)
	if len(platforms) == 0 || len(platformSKUs) == 0 {
		return result
	}
	var mappings []model.OrderSKUMapping
	if err := database.DB.Select("platform", "platform_sku", "amazon_sku").
		Where("platform IN ? AND platform_sku IN ? AND status = ? AND is_current = true", platforms, platformSKUs, "active").
		Find(&mappings).Error; err != nil {
		return result
	}
	for _, mapping := range mappings {
		result[mapping.Platform+"\x00"+mapping.PlatformSKU] = mapping.AmazonSKU
	}
	return result
}

func inferOrderCarrier(trackingNumber string) string {
	value := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(trackingNumber), " ", ""))
	if value == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(value, "AFTN"):
		return "Amazon MCF (TK)"
	case strings.HasPrefix(value, "TBA"):
		return "Amazon Logistics"
	case strings.HasPrefix(value, "GFUS"):
		return "GOFO"
	case strings.HasPrefix(value, "1Z"):
		return "UPS"
	case hasAnyPrefix(value, "9400", "9205", "9270", "9261", "9361", "9374", "420"):
		return "USPS"
	case len(value) == 13 && strings.HasSuffix(value, "US"):
		return "USPS"
	case hasAnyPrefix(value, "JJD", "JD"):
		return "DHL"
	case strings.HasPrefix(value, "YT"):
		return "YunExpress"
	case strings.HasPrefix(value, "SF"):
		return "SF Express"
	case isDigits(value) && (len(value) == 12 || len(value) == 15):
		return "FedEx"
	case isDigits(value) && len(value) == 10:
		return "DHL Express"
	default:
		return ""
	}
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func isDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return value != ""
}
