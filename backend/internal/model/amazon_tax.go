package model

type AmazonProductCategory struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Keyword  string `gorm:"size:200;not null" json:"keyword"`
	Category string `gorm:"size:100;not null" json:"category"`
}

// Amazon reports retain the source columns verbatim; computed periods are indexed separately.
type AmazonTaxShipment struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	ImportLogID uint              `gorm:"index;not null" json:"import_log_id"`
	StoreID     string            `gorm:"size:100;index" json:"store_id"`
	SiteID      string            `gorm:"size:20;index" json:"site_id"`
	Month       string            `gorm:"size:7;index;not null" json:"month"`
	Quarter     string            `gorm:"size:7;index;not null" json:"quarter"`
	Fingerprint string            `gorm:"size:64;index;not null" json:"-"`
	Data        map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"data"`
}

type AmazonTaxFinance struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	ImportLogID uint              `gorm:"index;not null" json:"import_log_id"`
	StoreID     string            `gorm:"size:100;index" json:"store_id"`
	SiteID      string            `gorm:"size:20;index" json:"site_id"`
	Month       string            `gorm:"size:7;index;not null" json:"month"`
	Quarter     string            `gorm:"size:7;index;not null" json:"quarter"`
	Fingerprint string            `gorm:"size:64;index" json:"-"`
	Data        map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"data"`
}
