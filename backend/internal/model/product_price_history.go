package model

import "time"

type ProductPriceHistory struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ProductSKUID    uint      `gorm:"column:product_sku_id;not null;uniqueIndex:idx_product_price_history_day" json:"product_sku_id"`
	SKU             string    `gorm:"type:varchar(120);not null;index" json:"sku"`
	PriceDate       time.Time `gorm:"type:date;not null;uniqueIndex:idx_product_price_history_day" json:"price_date"`
	AmazonPrice     *string   `gorm:"type:numeric(10,2)" json:"amazon_price"`
	AmazonSalePrice *string   `gorm:"type:numeric(10,2)" json:"amazon_sale_price"`
	AmazonBestDeal  *string   `gorm:"type:numeric(10,2)" json:"amazon_best_deal"`
	CreatedAt       time.Time `json:"created_at"`
}

func (ProductPriceHistory) TableName() string { return "product_price_histories" }
