package model

import "time"

// TikTokSKUPrice follows the platform SKU when its current Amazon mapping changes.
type TikTokSKUPrice struct {
	PlatformSKU string    `gorm:"type:varchar(160);primaryKey" json:"platform_sku"`
	PID         string    `gorm:"column:pid;type:varchar(32);not null;default:''" json:"pid"`
	Price       *string   `gorm:"type:numeric(10,2)" json:"price"`
	UpdatedBy   uint      `gorm:"not null;default:0" json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TikTokSKUPrice) TableName() string { return "tiktok_sku_prices" }
