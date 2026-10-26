package model

import "time"

type OrderSKUMapping struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Platform            string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_order_sku_mapping_platform_sku_amazon" json:"platform"`
	PlatformSKU         string    `gorm:"type:varchar(160);not null;uniqueIndex:idx_order_sku_mapping_platform_sku_amazon" json:"platform_sku"`
	AmazonSKU           string    `gorm:"type:varchar(160);not null;uniqueIndex:idx_order_sku_mapping_platform_sku_amazon;index" json:"amazon_sku"`
	NonAmazonSKU        string    `gorm:"type:varchar(160);not null;default:'';index" json:"non_amazon_sku"`
	Remark              string    `gorm:"type:text;not null;default:''" json:"remark"`
	IsCurrent           bool      `gorm:"not null;default:false;index" json:"is_current"`
	RelatedAmazonSKUs   []string  `gorm:"-" json:"related_amazon_skus"`
	CurrentAmazonSKU    string    `gorm:"-" json:"current_amazon_sku"`
	SelectionRequired   bool      `gorm:"-" json:"selection_required"`
	Status              string    `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	CreatedBy           uint      `gorm:"not null;default:0" json:"created_by"`
	UpdatedBy           uint      `gorm:"not null;default:0" json:"updated_by"`
	UpdatedByName       string    `gorm:"-" json:"updated_by_name"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (OrderSKUMapping) TableName() string {
	return "order_sku_mappings"
}
