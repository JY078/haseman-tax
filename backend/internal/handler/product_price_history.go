package handler

import (
	"net/http"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type productPriceHistoryRow struct {
	ID              uint      `json:"id"`
	SKU             string    `json:"sku"`
	Name            string    `json:"name"`
	ImageURL        string    `json:"image_url"`
	PriceDate       time.Time `json:"price_date"`
	AmazonPrice     *string   `json:"amazon_price"`
	AmazonSalePrice *string   `json:"amazon_sale_price"`
	AmazonBestDeal  *string   `json:"amazon_best_deal"`
}

func recordProductPriceHistory(tx *gorm.DB, item productPriceRow, now time.Time) error {
	day := now.UTC().Truncate(24 * time.Hour)
	history := model.ProductPriceHistory{ProductSKUID: item.ID, SKU: item.SKU, PriceDate: day, AmazonPrice: item.AmazonPrice, AmazonSalePrice: item.AmazonSalePrice, AmazonBestDeal: item.AmazonBestDeal}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "product_sku_id"}, {Name: "price_date"}}, DoUpdates: clause.Assignments(map[string]interface{}{"sku": item.SKU, "amazon_price": item.AmazonPrice, "amazon_sale_price": item.AmazonSalePrice, "amazon_best_deal": item.AmazonBestDeal})}).Create(&history).Error
}

func ListProductPriceHistory(c *gin.Context) {
	if _, ok := getOrderMappingUser(c); !ok {
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	query := database.DB.Table("product_price_histories AS h").Joins("JOIN product_skus AS p ON p.id = h.product_sku_id").Joins("JOIN product_spus AS s ON s.id = p.spu_id").Where("UPPER(TRIM(s.spu)) NOT LIKE 'CA%' AND UPPER(TRIM(s.spu)) NOT LIKE 'EU%' AND UPPER(TRIM(s.spu)) NOT LIKE 'UK%'").Select("h.id,h.sku,COALESCE(NULLIF(TRIM(p.name), ''), s.name) AS name,p.image_url,h.price_date,h.amazon_price,h.amazon_sale_price,h.amazon_best_deal").Order("h.price_date DESC, h.sku ASC")
	if from != "" {
		query = query.Where("h.price_date >= ?", from)
	}
	if to != "" {
		query = query.Where("h.price_date <= ?", to)
	}
	if id := c.Query("product_sku_id"); id != "" {
		query = query.Where("h.product_sku_id = ?", id)
	}
	var rows []productPriceHistoryRow
	if err := query.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取价格历史"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}
