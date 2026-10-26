package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type productPriceRow struct {
	ImageURL        string     `json:"image_url"`
	ID              uint       `json:"id"`
	SKU             string     `json:"sku"`
	ASIN            string     `json:"asin"`
	Name            string     `json:"name"`
	SPU             string     `json:"spu"`
	Color           string     `json:"color"`
	Size            string     `json:"size"`
	AmazonPrice     *string    `json:"amazon_price"`
	AmazonSalePrice *string    `json:"amazon_sale_price"`
	AmazonBestDeal  *string    `json:"amazon_best_deal"`
	PriceUpdatedAt  *time.Time `json:"price_updated_at"`
	MappingID       *uint      `json:"mapping_id"`
	PlatformSKU     *string    `json:"platform_sku"`
	PID             *string    `gorm:"column:pid" json:"pid"`
	TikTokPrice     *string    `gorm:"column:tiktok_price" json:"tiktok_price"`
	TikTokUpdatedAt *time.Time `gorm:"column:tiktok_updated_at" json:"tiktok_updated_at"`
}

func productPriceQuery() *gorm.DB {
	return database.DB.Table("product_skus AS p").
		Joins("JOIN product_spus AS s ON s.id = p.spu_id").
		Where("UPPER(TRIM(s.spu)) NOT LIKE 'CA%' AND UPPER(TRIM(s.spu)) NOT LIKE 'EU%' AND UPPER(TRIM(s.spu)) NOT LIKE 'UK%'").
		Joins("LEFT JOIN order_sku_mappings AS m ON m.amazon_sku = p.sku AND m.platform = 'TIKTOK' AND m.status = 'active' AND m.is_current = true").
		Joins("LEFT JOIN tiktok_sku_prices AS t ON t.platform_sku = m.platform_sku")
}

func ListProductPrices(c *gin.Context) {
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := min(parsePositiveInt(c.Query("page_size"), 20), 100)
	query := productPriceQuery()
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(p.sku) LIKE ? OR LOWER(p.asin) LIKE ? OR LOWER(COALESCE(NULLIF(TRIM(p.name), ''), s.name)) LIKE ? OR LOWER(s.spu) LIKE ? OR LOWER(m.platform_sku) LIKE ? OR t.pid LIKE ?", like, like, like, like, like, like)
	}
	switch c.Query("filter") {
	case "missing":
		// A deal listing stores only its deal price, so it is not incomplete.
		query = query.Where("(p.amazon_price IS NULL AND p.amazon_sale_price IS NULL AND p.amazon_best_deal IS NULL) OR (m.id IS NOT NULL AND (t.price IS NULL OR COALESCE(t.pid, '') = ''))")
	case "sale":
		query = query.Where("p.amazon_sale_price IS NOT NULL OR p.amazon_best_deal IS NOT NULL")
	case "unmapped":
		query = query.Where("m.id IS NULL")
	case "", "all":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的筛选条件"})
		return
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取价格记录"})
		return
	}
	items := []productPriceRow{}
	if err := query.Select("p.id, p.image_url, p.sku, p.asin, COALESCE(NULLIF(TRIM(p.name), ''), s.name) AS name, s.spu, p.color, p.size, p.amazon_price, p.amazon_sale_price, p.amazon_best_deal, p.price_updated_at, m.id AS mapping_id, m.platform_sku, t.pid, t.price AS tiktok_price, t.updated_at AS tiktok_updated_at").
		Order("CASE WHEN m.id IS NULL THEN 1 ELSE 0 END ASC, p.sku ASC, m.platform_sku ASC, p.id ASC, m.id ASC").Limit(pageSize).Offset((page - 1) * pageSize).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取价格记录"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize, "can_edit_tiktok": canEditOrderPlatform(user, "TIKTOK")})
}

var pricePattern = regexp.MustCompile(`^[0-9]{1,8}(\.[0-9]{1,2})?$`)
var pidPattern = regexp.MustCompile(`^[0-9]{1,32}$`)

// Empty prices remain NULL; zero is a recorded price, not a missing value.
func normalizeProductPrice(value *string) (*string, int64, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, 0, nil
	}
	raw := strings.TrimSpace(*value)
	if !pricePattern.MatchString(raw) {
		return nil, 0, errors.New("价格须为 0 至 99999999.99，最多两位小数")
	}
	parts := strings.SplitN(raw, ".", 2)
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	fraction := "00"
	if len(parts) == 2 {
		fraction = (parts[1] + "0")[:2]
	}
	minor, _ := strconv.ParseInt(fraction, 10, 64)
	valueNormalized := strconv.FormatInt(whole, 10) + "." + fraction
	return &valueNormalized, whole*100 + minor, nil
}

type productPriceInput struct {
	Price     *string `json:"price"`
	SalePrice *string `json:"sale_price"`
	BestDeal  *string `json:"best_deal"`
	PID       string  `json:"pid"`
}

func validateProductPriceInput(input *productPriceInput, platform string) error {
	var amount, sale int64
	var err error
	input.Price, amount, err = normalizeProductPrice(input.Price)
	if err != nil {
		return err
	}
	if platform == "amazon" {
		input.SalePrice, sale, err = normalizeProductPrice(input.SalePrice)
		if err != nil {
			return err
		}
		if input.SalePrice != nil && (input.Price == nil || sale > amount) {
			return errors.New("请填写原价，且折扣价不能高于原价")
		}
		input.BestDeal, _, err = normalizeProductPrice(input.BestDeal)
		if err != nil {
			return err
		}
	} else {
		input.PID = strings.TrimSpace(input.PID)
		if input.PID != "" && !pidPattern.MatchString(input.PID) {
			return errors.New("PID 须为不超过 32 位的数字，请勿填写完整链接")
		}
	}
	return nil
}

func UpdateProductPrice(c *gin.Context) {
	platform := c.Param("platform")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 || (platform != "amazon" && platform != "tiktok") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的商品或平台"})
		return
	}
	user, ok := getOrderMappingUser(c)
	if !ok {
		return
	}
	// Amazon prices inherit product catalog edit access; TikTok uses mapping permissions.
	if platform == "tiktok" && !canEditOrderPlatform(user, "TIKTOK") {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有 TikTok 平台编辑权限"})
		return
	}
	var input productPriceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的价格数据"})
		return
	}
	if err := validateProductPriceInput(&input, platform); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	if platform == "amazon" {
		result := database.DB.Model(&model.ProductSKU{}).Where("id = ?", id).Updates(map[string]interface{}{
			"amazon_price": input.Price, "amazon_sale_price": input.SalePrice, "amazon_best_deal": input.BestDeal, "price_updated_at": now, "updated_by": user.ID,
		})
		err = result.Error
		if err == nil && result.RowsAffected == 0 {
			err = gorm.ErrRecordNotFound
		}
		if err == nil {
			var product productPriceRow
			if readErr := database.DB.Table("product_skus AS p").Joins("JOIN product_spus AS s ON s.id = p.spu_id").Where("p.id = ?", id).Select("p.id,p.sku,COALESCE(NULLIF(TRIM(p.name), ''), s.name) AS name,p.image_url,p.amazon_price,p.amazon_sale_price,p.amazon_best_deal").Scan(&product).Error; readErr != nil {
				err = readErr
			} else {
				err = recordProductPriceHistory(database.DB, product, now)
			}
		}
	} else {
		err = database.DB.Transaction(func(tx *gorm.DB) error {
			var mapping model.OrderSKUMapping
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND platform = 'TIKTOK' AND status = 'active' AND is_current = true", id).First(&mapping).Error; err != nil {
				return err
			}
			item := model.TikTokSKUPrice{PlatformSKU: mapping.PlatformSKU, PID: input.PID, Price: input.Price, UpdatedBy: user.ID, UpdatedAt: now}
			return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "platform_sku"}}, DoUpdates: clause.AssignmentColumns([]string{"pid", "price", "updated_by", "updated_at"})}).Create(&item).Error
		})
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品或当前映射已变更，请刷新后重试"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "价格保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated_at": now})
}
