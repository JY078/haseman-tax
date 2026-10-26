package model

import "time"

type TikTokOrderBatch struct {
	ID        uint   `gorm:"primaryKey"`
	StoreID   string `gorm:"size:100;index;uniqueIndex:tiktok_order_batch_site_file"`
	SiteID    string `gorm:"size:20;index;uniqueIndex:tiktok_order_batch_site_file"`
	FileHash  string `gorm:"size:64;uniqueIndex:tiktok_order_batch_site_file;not null"`
	CreatedAt time.Time
}

// Keep the first imported row for each Order ID + SKU ID pair.
type TikTokOrder struct {
	ID      uint              `gorm:"primaryKey;index:idx_tiktok_order_lookup,priority:2,sort:desc" json:"id"`
	BatchID uint              `gorm:"index;not null" json:"batch_id"`
	StoreID string            `gorm:"size:100;index" json:"store_id"`
	SiteID  string            `gorm:"size:20;index" json:"site_id"`
	OrderID string            `gorm:"index:idx_tiktok_order_lookup,priority:1;not null" json:"order_id"`
	Month   string            `gorm:"size:7;index;not null" json:"month"`
	Quarter string            `gorm:"size:7;index;not null" json:"quarter"`
	Data    map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"data"`
}
