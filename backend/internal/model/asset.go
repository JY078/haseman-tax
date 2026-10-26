package model

import "time"

type Asset struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"type:varchar(255);not null;default:''" json:"name"`
	Type       string    `gorm:"type:varchar(80);not null;default:''" json:"type"`
	Category   string    `gorm:"type:varchar(120);not null;default:''" json:"category"`
	OwnerType  string    `gorm:"type:varchar(30);not null;default:'personal';index" json:"owner_type"`
	OwnerID    uint      `gorm:"not null;default:0;index" json:"owner_id"`
	UserID     uint      `gorm:"not null;default:0;index" json:"user_id"`
	Department string    `gorm:"type:varchar(100);not null;default:'';index" json:"department"`
	Visibility string    `gorm:"type:varchar(30);not null;default:'private';index" json:"visibility"`
	FileURL    string    `gorm:"type:varchar(1000);not null;default:''" json:"file_url"`
	CoverURL   string    `gorm:"type:varchar(1000);not null;default:''" json:"cover_url"`
	FileName   string    `gorm:"type:varchar(255);not null;default:''" json:"file_name"`
	FileType   string    `gorm:"type:varchar(80);not null;default:''" json:"file_type"`
	Tags       string    `gorm:"type:text;not null;default:'[]'" json:"tags"`
	UsageCount int       `gorm:"not null;default:0" json:"usage_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Asset) TableName() string {
	return "assets"
}

type AssetFavorite struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	SourceType string    `gorm:"type:varchar(80);not null" json:"source_type"`
	SourceID   string    `gorm:"type:varchar(120);not null" json:"source_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AssetFavorite) TableName() string {
	return "asset_favorites"
}

type AssetCategory struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"type:varchar(120);not null;default:''" json:"name"`
	Type       string    `gorm:"type:varchar(80);not null;default:''" json:"type"`
	OwnerType  string    `gorm:"type:varchar(30);not null;default:'personal';index" json:"owner_type"`
	OwnerID    uint      `gorm:"not null;default:0;index" json:"owner_id"`
	UserID     uint      `gorm:"not null;default:0;index" json:"user_id"`
	Department string    `gorm:"type:varchar(100);not null;default:'';index" json:"department"`
	Sort       int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (AssetCategory) TableName() string {
	return "asset_categories"
}
