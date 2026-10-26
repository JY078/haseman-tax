package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type orderFieldDefinition struct {
	FieldKey      string     `json:"field_key"`
	SourceHeaders []string   `json:"source_headers"`
	Required      bool       `json:"required"`
	UpdatedBy     uint       `json:"updated_by,omitempty"`
	UpdatedByName string     `json:"updated_by_name,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type orderFieldMappingsInput struct {
	Items []orderFieldMappingInputItem `json:"items"`
}

type orderFieldMappingInputItem struct {
	FieldKey      string   `json:"field_key"`
	SourceHeaders []string `json:"source_headers"`
}

var defaultOrderFieldMappings = map[string][]orderFieldDefinition{
	"TEMU": {
		{FieldKey: "orderId", SourceHeaders: []string{"订单号"}, Required: true},
		{FieldKey: "status", SourceHeaders: []string{"订单状态"}, Required: true},
		{FieldKey: "quantity", SourceHeaders: []string{"应履约件数"}, Required: true},
		{FieldKey: "productName", SourceHeaders: []string{"商品名称"}},
		{FieldKey: "sku", SourceHeaders: []string{"SKU货号"}, Required: true},
		{FieldKey: "recipient", SourceHeaders: []string{"收货人姓名"}, Required: true},
		{FieldKey: "phone", SourceHeaders: []string{"收货人联系方式"}, Required: true},
		{FieldKey: "email", SourceHeaders: []string{"邮箱"}},
		{FieldKey: "address1", SourceHeaders: []string{"详细地址1"}, Required: true},
		{FieldKey: "address2", SourceHeaders: []string{"详细地址2"}},
		{FieldKey: "address3", SourceHeaders: []string{"详细地址3"}},
		{FieldKey: "district", SourceHeaders: []string{"区县"}},
		{FieldKey: "city", SourceHeaders: []string{"城市"}, Required: true},
		{FieldKey: "state", SourceHeaders: []string{"省份"}, Required: true},
		{FieldKey: "postalCode", SourceHeaders: []string{"收货地址邮编"}, Required: true},
		{FieldKey: "country", SourceHeaders: []string{"国家"}, Required: true},
		{FieldKey: "createdTime", SourceHeaders: []string{"订单创建时间"}, Required: true},
	},
	"TIKTOK": {
		{FieldKey: "orderId", SourceHeaders: []string{"Order ID"}, Required: true},
		{FieldKey: "status", SourceHeaders: []string{"Order SubStatus"}, Required: true},
		{FieldKey: "sku", SourceHeaders: []string{"Seller SKU"}, Required: true},
		{FieldKey: "quantity", SourceHeaders: []string{"Quantity"}, Required: true},
		{FieldKey: "returnQuantity", SourceHeaders: []string{"Sku Quantity of return"}},
		{FieldKey: "fulfillmentType", SourceHeaders: []string{"Delivery Option Type"}, Required: true},
		{FieldKey: "productName", SourceHeaders: []string{"Product Name"}},
		{FieldKey: "recipient", SourceHeaders: []string{"Recipient"}, Required: true},
		{FieldKey: "phone", SourceHeaders: []string{"Phone #", "PHONE", "Phone"}, Required: true},
		{FieldKey: "country", SourceHeaders: []string{"Country"}, Required: true},
		{FieldKey: "state", SourceHeaders: []string{"State"}, Required: true},
		{FieldKey: "city", SourceHeaders: []string{"City"}, Required: true},
		{FieldKey: "postalCode", SourceHeaders: []string{"Zipcode"}, Required: true},
		{FieldKey: "address1", SourceHeaders: []string{"Address Line 1"}, Required: true},
		{FieldKey: "address2", SourceHeaders: []string{"Address Line 2"}},
		{FieldKey: "createdTime", SourceHeaders: []string{"Created Time"}, Required: true},
	},
	"LOWES": {
		{FieldKey: "orderId", SourceHeaders: []string{"Order number"}, Required: true},
		{FieldKey: "status", SourceHeaders: []string{"Status"}, Required: true},
		{FieldKey: "quantity", SourceHeaders: []string{"Quantity"}, Required: true},
		{FieldKey: "productName", SourceHeaders: []string{"Details"}},
		{FieldKey: "sku", SourceHeaders: []string{"Store SKU", "Offer SKU"}, Required: true},
		{FieldKey: "recipientFirstName", SourceHeaders: []string{"Shipping address first name"}, Required: true},
		{FieldKey: "recipientLastName", SourceHeaders: []string{"Shipping address last name"}},
		{FieldKey: "phone", SourceHeaders: []string{"Shipping address phone"}, Required: true},
		{FieldKey: "email", SourceHeaders: []string{"Shipping address email"}},
		{FieldKey: "address1", SourceHeaders: []string{"Shipping address street 1"}, Required: true},
		{FieldKey: "address2", SourceHeaders: []string{"Shipping address street 2"}},
		{FieldKey: "city", SourceHeaders: []string{"Shipping address city"}, Required: true},
		{FieldKey: "state", SourceHeaders: []string{"Shipping address state"}, Required: true},
		{FieldKey: "postalCode", SourceHeaders: []string{"Shipping address zip"}, Required: true},
		{FieldKey: "country", SourceHeaders: []string{"Shipping address country"}, Required: true},
		{FieldKey: "createdTime", SourceHeaders: []string{"Date created"}, Required: true},
	},
}

func GetOrderFieldMappings(c *gin.Context) {
	platform := normalizeOrderPlatform(c.Param("platform"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}

	items, err := loadOrderFieldMappings(platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch field mappings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"platform": platform,
		"items":    items,
		"can_edit": canEditOrderPlatform(user, platform),
	})
}

func ReplaceOrderFieldMappings(c *gin.Context) {
	platform := normalizeOrderPlatform(c.Param("platform"))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	if !canEditOrderPlatform(user, platform) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You cannot edit field mappings for this platform"})
		return
	}

	var input orderFieldMappingsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	definitions := make([]orderFieldDefinition, 0, len(input.Items))
	for _, item := range input.Items {
		definitions = append(definitions, orderFieldDefinition{
			FieldKey:      item.FieldKey,
			SourceHeaders: item.SourceHeaders,
		})
	}
	items, err := validateOrderFieldMappings(platform, definitions)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			headersJSON, marshalErr := json.Marshal(item.SourceHeaders)
			if marshalErr != nil {
				return marshalErr
			}
			row := model.OrderFieldMapping{
				Platform:      platform,
				FieldKey:      item.FieldKey,
				SourceHeaders: string(headersJSON),
				UpdatedBy:     user.ID,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if createErr := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "platform"}, {Name: "field_key"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"source_headers": row.SourceHeaders,
					"updated_by":     user.ID,
					"updated_at":     now,
				}),
			}).Create(&row).Error; createErr != nil {
				return createErr
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save field mappings"})
		return
	}
	savedItems, loadErr := loadOrderFieldMappings(platform)
	if loadErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Field mappings saved but could not be reloaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"platform": platform, "items": savedItems})
}

func loadOrderFieldMappings(platform string) ([]orderFieldDefinition, error) {
	defaults := defaultOrderFieldMappings[platform]
	items := make([]orderFieldDefinition, len(defaults))
	copy(items, defaults)

	var rows []model.OrderFieldMapping
	if err := database.DB.Where("platform = ?", platform).Find(&rows).Error; err != nil {
		return nil, err
	}
	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UpdatedBy)
	}
	names := loadUserDisplayNames(userIDs)
	type storedFieldMapping struct {
		headers       []string
		updatedBy     uint
		updatedByName string
		updatedAt     time.Time
	}
	stored := make(map[string]storedFieldMapping, len(rows))
	for _, row := range rows {
		var headers []string
		if json.Unmarshal([]byte(row.SourceHeaders), &headers) == nil {
			stored[row.FieldKey] = storedFieldMapping{
				headers:       headers,
				updatedBy:     row.UpdatedBy,
				updatedByName: names[row.UpdatedBy],
				updatedAt:     row.UpdatedAt,
			}
		}
	}
	for index := range items {
		if storedItem, exists := stored[items[index].FieldKey]; exists {
			if !isLegacyTikTokDefaultMapping(platform, items[index].FieldKey, storedItem.headers) {
				items[index].SourceHeaders = storedItem.headers
			}
			items[index].UpdatedBy = storedItem.updatedBy
			items[index].UpdatedByName = storedItem.updatedByName
			updatedAt := storedItem.updatedAt
			items[index].UpdatedAt = &updatedAt
		}
	}
	return items, nil
}

func isLegacyTikTokDefaultMapping(platform, fieldKey string, headers []string) bool {
	if platform != "TIKTOK" || len(headers) != 1 {
		return false
	}
	return (fieldKey == "status" && headers[0] == "Order Status") ||
		(fieldKey == "fulfillmentType" && headers[0] == "Fulfillment Type")
}

func validateOrderFieldMappings(platform string, input []orderFieldDefinition) ([]orderFieldDefinition, error) {
	defaults := defaultOrderFieldMappings[platform]
	allowed := make(map[string]orderFieldDefinition, len(defaults))
	for _, item := range defaults {
		allowed[item.FieldKey] = item
	}
	provided := make(map[string][]string, len(input))
	for _, item := range input {
		definition, exists := allowed[item.FieldKey]
		if !exists {
			return nil, newOrderMappingError("Unknown field: " + item.FieldKey)
		}
		if _, exists := provided[item.FieldKey]; exists {
			return nil, newOrderMappingError("Duplicate field: " + item.FieldKey)
		}
		headers := compactOrderSKUs(item.SourceHeaders)
		if definition.Required && len(headers) == 0 {
			return nil, newOrderMappingError("Required field has no source header: " + item.FieldKey)
		}
		provided[item.FieldKey] = headers
	}

	result := make([]orderFieldDefinition, 0, len(defaults))
	for _, definition := range defaults {
		headers, exists := provided[definition.FieldKey]
		if !exists {
			return nil, newOrderMappingError("Missing field: " + definition.FieldKey)
		}
		definition.SourceHeaders = headers
		result = append(result, definition)
	}
	return result, nil
}

type orderMappingError string

func (err orderMappingError) Error() string { return string(err) }

func newOrderMappingError(message string) error {
	return orderMappingError(strings.TrimSpace(message))
}
