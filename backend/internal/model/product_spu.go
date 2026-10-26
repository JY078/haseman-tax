package model

import "time"

type ProductSPU struct {
	ID            uint                   `gorm:"primaryKey" json:"id"`
	SPU           string                 `gorm:"type:varchar(120);not null;uniqueIndex" json:"spu"`
	Name          string                 `gorm:"type:varchar(255);not null;default:''" json:"name"`
	Brand         string                 `gorm:"type:varchar(120);not null;default:''" json:"brand"`
	Model         string                 `gorm:"type:varchar(120);not null;default:''" json:"model"`
	Material      string                 `gorm:"type:varchar(255);not null;default:''" json:"material"`
	Category1     string                 `gorm:"type:varchar(255);not null;default:''" json:"category1"`
	Category2     string                 `gorm:"type:varchar(255);not null;default:''" json:"category2"`
	ModelURL      string                 `gorm:"type:varchar(500);not null;default:''" json:"model_url"`
	UpdatedBy     uint                   `gorm:"not null;default:0" json:"updated_by"`
	UpdatedByName string                 `gorm:"-" json:"updated_by_name"`
	SKUs          []ProductSKU           `gorm:"foreignKey:SPUID;references:ID" json:"skus,omitempty"`
	Attachments   []ProductSPUAttachment `gorm:"foreignKey:SPUID;references:ID" json:"attachments,omitempty"`
	SKUCount      int                    `gorm:"-" json:"sku_count"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func (ProductSPU) TableName() string {
	return "product_spus"
}
