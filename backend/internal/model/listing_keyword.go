package model

import "time"

// ListingKeyword stores a reusable marketplace keyword through its lifecycle.
// Candidate records are kept so operators can review what the agents discover
// before a term becomes part of the active library.
type ListingKeyword struct {
	ID                uint64      `gorm:"primaryKey" json:"id"`
	Platform          string      `gorm:"type:varchar(50);not null;uniqueIndex:idx_listing_keywords_scope_term" json:"platform"`
	CountryCode       string      `gorm:"type:varchar(20);not null;default:'';uniqueIndex:idx_listing_keywords_scope_term" json:"country_code"`
	Category          string      `gorm:"type:varchar(200);not null;default:'';uniqueIndex:idx_listing_keywords_scope_term" json:"category"`
	Term              string      `gorm:"type:varchar(255);not null;uniqueIndex:idx_listing_keywords_scope_term" json:"term"`
	Stage             string      `gorm:"type:varchar(20);not null;default:'expanded'" json:"stage"`
	Status            string      `gorm:"type:varchar(20);not null;default:'candidate';index" json:"status"`
	Tags              StringSlice `gorm:"type:jsonb;serializer:json" json:"tags"`
	Source            string      `gorm:"type:varchar(40);not null;default:'operator'" json:"source"`
	SearchVolume      int         `gorm:"not null;default:0" json:"search_volume"`
	Competition       float64     `gorm:"not null;default:0" json:"competition"`
	PlatformFrequency int         `gorm:"not null;default:0" json:"platform_frequency"`
	SEOScore          int         `gorm:"not null;default:0" json:"seo_score"`
	FactIDs           StringSlice `gorm:"type:jsonb;serializer:json" json:"fact_ids"`
	EvidenceURLs      StringSlice `gorm:"type:jsonb;serializer:json" json:"evidence_urls"`
	UsageCount        int         `gorm:"not null;default:0" json:"usage_count"`
	Notes             string      `gorm:"type:text;not null;default:''" json:"notes"`
	LastUsedAt        *time.Time  `json:"last_used_at,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

func (ListingKeyword) TableName() string { return "listing_keywords" }
