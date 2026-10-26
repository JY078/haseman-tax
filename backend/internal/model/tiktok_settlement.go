package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type TikTokSettlementBatch struct {
	ID        uint   `gorm:"primaryKey"`
	StoreID   string `gorm:"size:100;index;uniqueIndex:tiktok_settlement_batch_site_file"`
	SiteID    string `gorm:"size:20;index;uniqueIndex:tiktok_settlement_batch_site_file"`
	FileHash  string `gorm:"size:64;uniqueIndex:tiktok_settlement_batch_site_file;not null"`
	CreatedAt time.Time
}

// Preserve the original fields, including IDs and blank amounts, without float conversion.
type TikTokSettlement struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	BatchID         uint              `gorm:"index;not null" json:"batch_id"`
	StoreID         string            `gorm:"size:100;index" json:"store_id"`
	SiteID          string            `gorm:"size:20;index" json:"site_id"`
	Month           string            `gorm:"size:7;index;not null" json:"month"`
	Quarter         string            `gorm:"size:7;index;not null" json:"quarter"`
	StatementDate   string            `gorm:"size:10;index;not null" json:"statement_date"`
	LogisticsMethod string            `json:"logistics_method"`
	OrderType       string            `json:"order_type"`
	Currency        string            `gorm:"size:3" json:"currency"`
	Amount          decimal.Decimal   `gorm:"type:numeric(24,8)" json:"-"`
	Data            map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"data"`
}
